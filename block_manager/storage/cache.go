package storage

import (
	"github.com/secnot/gobalance/primitives"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	InitialQueueSize = 10000
)

type StorageCache struct {
	// 
	sto Storage

	// pending inserts
	inserts map[TxOutId]TxOutData

	// pending deletions
	deletions map[TxOutId]bool

	// uncommited txouts address balance
	balance map[string]int64

	// Heigh and has for the last block in cache (NOT THE SAME AS STORED)
	height int64

	lastBlockHash chainhash.Hash
	
	//
	balanceIndexEnabled bool
}

// NewStorageCache creates a new cache, with or without balance indexing
func NewStorageCache(sto Storage, balanceIndex bool) (s *StorageCache, err error) {
	height, hash, err := sto.GetLastBlock()
	if err != nil {
		return nil, err
	}

	cache := StorageCache{
		sto:       sto,
		inserts:   make(map[TxOutId]TxOutData, InitialQueueSize),
		deletions: make(map[TxOutId]bool, InitialQueueSize),
		balance :  make(map[string]int64, InitialQueueSize),
		height:    height,
		lastBlockHash:       hash,
		balanceIndexEnabled: balanceIndex,
	}

	return &cache, nil
}


// Len returns the number of stored utxo - deletions + inserts
func (s *StorageCache) Len() (length int, err error){
	length, err = s.sto.Len()
	if err != nil {
		return
	}

	return length, nil
}

// UncommitedLen returns the number of uncommitted inserts+deletions
func (s *StorageCache) UncommittedLen() (size int) {
	return len(s.inserts) + len(s.deletions)
}

// SetHeight sets new storage height
func (s *StorageCache) SetHeight(height int64) {
	s.height = height
}

// GetHeight
func (s *StorageCache) GetHeight() int64 {
	return s.height
}

// SetHash sets last block hash
func (s *StorageCache) SetHash(hash chainhash.Hash) {
	s.lastBlockHash = hash
}

// GetHash returns last block hash
func (s *StorageCache) GetHash() chainhash.Hash {
	return s.lastBlockHash
}

// GetTxOut
func (s *StorageCache) GetTxOut(id TxOutId) (utxo TxOutData, err error){

	// if id is slated for deletion it isn't available
	if _, ok := s.deletions[id]; ok {
		return TxOutData{Addr: "", Value: 0}, nil
	}

	// Search into local cache
	if data, ok := s.inserts[id]; ok {
		return data, nil
	}

	// Fetch from storage
	return s.sto.Get(id)
}

// Contains returns true if the storage or cache contains the TxOut
func (s *StorageCache) Contains(id TxOutId) (bool, error) {

	// if id is slated for deletion it isn't contained
	if _, ok := s.deletions[id]; ok {
		return false, nil
	}

	// id is queued for insertion into storage
	if _, ok := s.inserts[id]; ok {
		return true, nil
	}

	// Check storage
	return s.sto.Contains(id)
}

// BulkGetTxOut
func (s *StorageCache) BulkGetTxOut(ids []TxOutId) (outs []TxOutData, err error) {

	outs     = make([]TxOutData, len(ids))
	missing := make([]TxOutId, 0, len(ids)) // Ids not cached

	// Find cached TxOuts
	for n, id := range ids {
		if data, ok := s.inserts[id]; ok {
			outs[n] = data
			continue
		}

		missing = append(missing, id)
	}

	// Fetch missing from storage
	if len(missing) > 0 {

		fetched, err := s.sto.BulkGet(missing)
		if err != nil {
			return nil, err
		}

		// Place them in the correct position and return
		i := 0 // 
		for n, id := range ids {
			if id == missing[i] {
				outs[n] = fetched[i]	
				i += 1
			}
			if i >= len(missing) {
				break
			}
		}
	}

	return outs, nil
}

// updateBalance
func (s *StorageCache) updateBalance(address string, balance int64) {

	if !s.balanceIndexEnabled {
		return
	}

	new_balance := s.balance[address] + balance
	if new_balance == 0 {
		delete(s.balance, address)
	} else {
		s.balance[address] = new_balance
	}
}

// AddTxOut queues a TxOut for insertion into storage (without updating balance index
func (s *StorageCache) addTxOut(utxo primitives.TxOut) {
	
	id   := TxOutId{TxHash: *utxo.TxHash, Nout: utxo.Nout}
	if _, ok := s.deletions[id]; ok {
		delete(s.deletions, id)
		return
	}
	
	s.inserts[id] = TxOutData{Addr: utxo.Addr, Value: utxo.Value}
}

// DelTxOut queues TxOutId for deletion from storage
func (s *StorageCache) delTxOut(id TxOutId) {
	
	// If utxo is a pending insert discard it and return
	if _, ok := s.inserts[id]; ok {
		delete(s.inserts, id)
		return
	}

	// Otherwise add to pending deletions to delete it from storage
	s.deletions[id] = true
}

// AddBlock adds block transaction outputs and delete its inputs
func (s *StorageCache) AddBlock(block *primitives.Block) error {

	for _, tx := range block.Transactions {
	
		// Add transaction outputs
		for _, out := range tx.Out {
			if out.Addr != "" && out.Value != 0 {
				s.addTxOut(*out)
				s.updateBalance(out.Addr, out.Value)
			}
		}

		// Delete transaction inputs
		for _, in := range tx.In {
			if in.Addr != "" && in.Value != 0 {
				s.delTxOut(TxOutId{TxHash: *in.TxHash, Nout: in.Nout})
				s.updateBalance(in.Addr, -in.Value)
			}
		}
	}
		
	// Update height
	s.SetHeight(int64(block.Height))
	s.SetHash(block.Hash)
	return nil
}

// GetBalance returns the address balance
func (s *StorageCache) GetBalance(address string) (int64, error) {
	storedBalance, err := s.sto.GetBalance(address)
	if err != nil {
		return -1, nil
	}
	
	cachedBalance := s.balance[address]
	return cachedBalance+storedBalance, nil
}

// Commit pending insertion, deletions, and height into storage
func (s *StorageCache) Commit() (err error){

	// Update DB
	err = s.sto.BulkUpdateFromMap(s.inserts, s.deletions, s.height, s.lastBlockHash)
	if err != nil {
		return err
	}

	// Re allocate cache maps after successfull commit
	s.inserts   = make(map[TxOutId]TxOutData, InitialQueueSize)
	s.deletions = make(map[TxOutId]bool, InitialQueueSize)
	s.balance   = make(map[string]int64, InitialQueueSize)

	// Clean inserts and deletions
	return nil
}

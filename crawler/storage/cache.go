package storage

import (
	"github.com/secnot/simplelru"
	"github.com/secnot/gobalance/primitives"
)


const (
	// Initial size for insertion and deletions queue
	InitialQueueSize = 10000
)

type StorageCache struct {
	// 
	sto Storage

	// Storage cache TxOutId -> TxOutData
	cache *simplelru.LRUCache

	// pending inserts
	inserts map[TxOutId]TxOutData

	// pending deletions
	deletions map[TxOutId]bool

	// This is the proxy height, not the stored height
	height int64
}


func NewStorageCache(sto Storage, cache_size int) (s *StorageCache, err error) {
	height, err := sto.GetHeight()
	if err != nil {
		return nil, err
	}

	cache := StorageCache{
		sto: sto,
		cache: simplelru.NewLRUCache(cache_size, 10),
		inserts: make(map[TxOutId]TxOutData, InitialQueueSize),
		deletions: make(map[TxOutId]bool, InitialQueueSize),
		height: height,
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

// CleanCache
func (s *StorageCache) CleanCache() {
	s.cache.Purge()
}

// SetHeight sets new storage height
func (s *StorageCache) SetHeight(height int64) {
	s.height = height
}

// GetHeight
func (s *StorageCache) GetHeight() int64 {
	return s.height
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

	if data, ok := s.cache.Get(id); ok {
		return data.(TxOutData), nil
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

		if data, ok := s.cache.Get(id); ok {
			outs[n] = data.(TxOutData)
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

// AddTxOut queues a TxOut for insertion into storage
func (s *StorageCache) AddTxOut(utxo primitives.TxOut) {
	data := TxOutData{Addr: utxo.Addr, Value: utxo.Value}
	id   := TxOutId{TxHash: *utxo.TxHash, Nout: utxo.Nout}
	s.inserts[id] = data
}

// DelTxOut queues TxOutId for deletion from storage
func (s *StorageCache) DelTxOut(id TxOutId) {
	
	// If utxo is a pending insert discard it and return
	if _, ok := s.inserts[id]; ok {
		delete(s.inserts, id)
		return
	}

	// Otherwise add to pending deletions
	s.deletions[id] = true
}

// Commit pending insertion, deletions, and height into storage
func (s *StorageCache) Commit() (err error){

	// 1 - Commit inserts/deletions to db
	toInsert := make([]primitives.TxOut, 0, len(s.inserts))
	toDelete := make([]TxOutId, 0, len(s.deletions))
	
	for id, data := range s.inserts {
		hash := id.TxHash
		utxo := primitives.TxOut {
			TxHash: &hash,
			Nout: id.Nout,
			Addr: data.Addr,
			Value: data.Value,
		}
		toInsert = append(toInsert, utxo)
	}

	for id, _ := range s.deletions {
		toDelete = append(toDelete, id)	
	}

	err = s.sto.BulkUpdate(toInsert, toDelete, s.height)
	if err != nil {
		return err
	}

	// Remove deleted from cache
	for id, _ := range s.deletions {
		s.cache.Remove(id)
	}

	// Cache stored utxo
	for id, data := range s.inserts {
		s.cache.Set(id, data)
	}

	// Clean inserts and deletions
	s.inserts   = make(map[TxOutId]TxOutData, InitialQueueSize)
	s.deletions = make(map[TxOutId]bool, InitialQueueSize)

	return nil
}

// Resize cache max size.
func (s *StorageCache) Resize(size int) {
	s.cache.Resize(size, 10)
}

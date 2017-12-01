package block_manager


import (
	"log"
	"time"
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/secnot/gobalance/primitives"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/secnot/gobalance/block_manager/storage"
)

const (
	// Average number of transaction inputs in a block
	AverageBlockInputs = 2048 * 10 // Transactions * Inputs
)

var ErrBacktrackLimit = errors.New("Backtrack limit reached")

type BlockManager struct {

	// Last block height
	height int64

	// Last time a block was added 
	lastTime time.Time

	// Max number of txout cached in memory before a commit is forced
	commitSize int

	// 
	storageCache *storage.StorageCache

	// Confirmations required for a block to be elegible for storage
	confirmations uint16

	// Blocks waiting for enough confirmations before committing to storage
	pendingBlocks *primitives.BlockQueue

	// Sync mode flag, when in sync mode the balance is disabled to save memory
	// and for a faster syncing
	sync bool
}

// NewBlockManager
func NewBlockManager(sto storage.Storage, cacheSize int, confirmations uint16, sync bool) (*BlockManager, error) {

	cache, err := storage.NewStorageCache(sto, !sync)
	if err != nil {
		return nil, err
	}
	
	manager := BlockManager {
		height: 		cache.GetHeight(),
		lastTime:       time.Now(),
		storageCache:	cache,
		commitSize:     cacheSize,
		confirmations:	confirmations,
		pendingBlocks:	primitives.NewBlockQueue(),
		sync:           sync,
	}

	return &manager, nil
}

// getBlockIntpus populates transactions inputs address and value
func (b *BlockManager) getBlockInputs(block *primitives.Block) (*primitives.Block, error) {	
	
	// map all the block transactions
	blockTxIdx := make(map[chainhash.Hash]*primitives.Tx, len(block.Transactions))
	for _, tx := range block.Transactions {
		blockTxIdx[*tx.Hash] = tx
	}
	
	// Find all the missing transaction inputs to retrieve them in a single bulk 
	// operartion
	missingIns := make([]storage.TxOutId, 0, AverageBlockInputs)
	
	for _, tx := range block.Transactions {

		if tx.IsCoinBase() {
			continue
		}

		// Add to missing inputs all the ones not queued or from the current block
		for _, txIn := range tx.In {

			txInBlock := blockTxIdx[*txIn.TxHash]
			txInQueue, _ := b.pendingBlocks.Tx(*txIn.TxHash)

			if txInBlock == nil && txInQueue == nil {
				missingIns = append(missingIns, storage.TxOutId{
					TxHash: *txIn.TxHash, Nout: txIn.Nout,})
			}
		}
	}

	// Get missing transaction outputs from storage
	missingData, err := b.storageCache.BulkGetTxOut(missingIns)
	if err != nil {
		return nil, err
	}

	// Add data to transactions inputs
	var missingIdx int = 0 // Index for the next unused missing input

	for _, tx := range block.Transactions {
	
		if tx.IsCoinBase() {
			continue
		}

		for _, txIn := range tx.In {
			if tx, _ := b.pendingBlocks.Tx(*txIn.TxHash); tx != nil {
				// The input is an output from a pending block
				txIn.Addr  = tx.Out[txIn.Nout].Addr
				txIn.Value = tx.Out[txIn.Nout].Value
			} else if tx := blockTxIdx[*txIn.TxHash]; tx != nil {
				// The input is an output from the current block
				txIn.Addr  = tx.Out[txIn.Nout].Addr
				txIn.Value = tx.Out[txIn.Nout].Value
			} else {
				// The input was retrieved from storage
				txIn.Addr  = missingData[missingIdx].Addr
				txIn.Value = missingData[missingIdx].Value
				missingIdx += 1
			}
		}
	}	

	return block, nil
}

// buildTx returns a primitives.Tx for the MsgTx (without inputs addresses and values)
func (b *BlockManager) buildTx(wireTx *wire.MsgTx) *primitives.Tx {
	hash := wireTx.TxHash()
	tx := primitives.NewTx(&hash)

	for nout, txOut := range wireTx.TxOut {
		address := primitives.PkScriptToAddr(txOut.PkScript)
		txout := primitives.NewTxOut(&hash, uint32(nout), address, txOut.Value)
		if txout != nil {
			tx.AddOut(txout)
		}
	}

	for _, txIn := range wireTx.TxIn {
		prevOutHash := txIn.PreviousOutPoint.Hash
		txin := primitives.NewTxOut(&prevOutHash, uint32(txIn.PreviousOutPoint.Index), "", 0)
		if txin != nil {
			tx.AddIn(txin)
		}
	}

	return tx
}

func (b *BlockManager) buildBlock(bHash *chainhash.Hash, block *wire.MsgBlock, height uint64) (*primitives.Block, error) {
	
	// Build all the block transactions (without inputs)
	transactions := make([]*primitives.Tx, 0, len(block.Transactions))
	for _, wireTx := range block.Transactions {
		tx := b.buildTx(wireTx)
		transactions = append(transactions, tx)
	}

	pBlock := primitives.NewBlock(*bHash, block.Header.PrevBlock, height)
	pBlock.Transactions = transactions

	return pBlock, nil
}

// TimeSinceLastBlock returns the seconds elapsed since the last block was processed
func (b *BlockManager) TimeSinceLastBlock() float64 {
	now := time.Now()
	return now.Sub(b.lastTime).Seconds() // time since last block
}

// AddBlock adds a wire.Block to the manager returning primitives.Block equivalent
func (b *BlockManager) AddBlock(block *wire.MsgBlock, blockHash *chainhash.Hash) (*primitives.Block, error) {
	
	// Generate	block and add to pending
	pBlock, err := b.buildBlock(blockHash, block, uint64(b.height+1))
	if err != nil {
		return nil, err
	}

	if !b.sync {
		pBlock, err = b.getBlockInputs(pBlock)
		if err != nil {
			return nil, err
		}
	}

	b.pendingBlocks.PushBack(pBlock)

	// Update current height
	b.height += 1

	// Add confirmed block and height to storage cache
	if b.pendingBlocks.Len() > int(b.confirmations) {
		confirmedBlock := b.pendingBlocks.PopFront()
		b.storageCache.AddBlock(confirmedBlock)
	}

	// Commit cache when there are enough changes
	if b.storageCache.UncommittedLen() > b.commitSize || b.TimeSinceLastBlock() > 120.0 {
		
		log.Print("Commit: ", b.storageCache.GetHeight())
		err := b.storageCache.Commit()
		if err != nil {
			return nil, err
		}
	}
	b.lastTime = time.Now()

	return pBlock, nil
}

// BacktrackBlock backtracks and returns last block
func (b *BlockManager) BacktrackBlock() (*primitives.Block, error){
	if b.pendingBlocks.Len() == 0 {
		return nil, ErrBacktrackLimit
	}

	b.height -= 1
	block := b.pendingBlocks.PopBack()
	return block, nil
}

// GetHeight returns the last block height
func (b *BlockManager) Height() int64 {
	return b.height
}

// GetBalance returns address balance
func (b *BlockManager) GetBalance(address string) (int64, error) {
	pendingBalance, ok := b.pendingBlocks.GetBalance(address)
	if !ok {
		pendingBalance = 0
	}
	storedBalance, err  := b.storageCache.GetBalance(address)
	if err != nil  {
		return 0, err
	}

	return pendingBalance + storedBalance, nil
}

// Synced returns true when the manager is with the last blockchain block
func (b *BlockManager) Synced() bool {
	return b.TimeSinceLastBlock() > 30.0
}

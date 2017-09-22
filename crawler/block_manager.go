package crawler


import (
	"log"
	"time"
	_ "sync"
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/crawler/storage"
)

const (
	UtxoCacheSize = 800000
	
	// Average number of transaction inputs in a block
	AverageBlockInputs = 2048 * 10 // Transactions * Inputs

	// Number of pending commits ve
	StorageCommitSize = 10000000
)

var ErrBacktrackLimit = errors.New("not found")


type BlockManager struct {

	// Last block height
	height int64

	// Last time a block was added 
	lastTime time.Time

	// 
	storageCache *storage.StorageCache

	// Confirmations required for a block to be elegible for storage
	confirmations uint16

	// Blocks waiting for enough confirmations before committing to storage
	pendingBlocks *primitives.BlockQueue
}

// NewBlockManager
func NewBlockManager(sto storage.Storage, cacheSize int, confirmations uint16) (*BlockManager, error) {

	cache, err := storage.NewStorageCache(sto, cacheSize)
	if err != nil {
		return nil, err
	}
	
	manager := BlockManager {
		height: 		cache.GetHeight(),
		lastTime:       time.Now(),
		storageCache:	cache,
		confirmations:	confirmations,
		pendingBlocks:	primitives.NewBlockQueue(),
	}

	return &manager, nil
}

// buildTx returns a primitives.Tx for the MsgTx (excluding the inputs)
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

	return tx
}

// buildBlock constructs a primitive.Block from wire.MsgBlock.
func (b *BlockManager) buildBlock(bHash *chainhash.Hash, block *wire.MsgBlock, height uint64) (*primitives.Block, error) {
		
	// Block construction is a little convoluted to optimize storage IO by 
	// loading all the missing transactions inputs using a single BulkGet operation.
	
	// Build all the block transactions (without inputs)
	transactions := make([]*primitives.Tx, 0, len(block.Transactions))
	blockTxIdx := make(map[chainhash.Hash]*primitives.Tx, len(block.Transactions))
	for _, wireTx := range block.Transactions {
		tx := b.buildTx(wireTx)
		transactions = append(transactions, tx)
		blockTxIdx[*tx.Hash] = tx
	}
	
	// Find all the transaction inputs that need to be retrieved from storage
	missingIns := make([]storage.TxOutId, 0, AverageBlockInputs)
	
	for _, wireTx := range block.Transactions {
		for _, txIn := range wireTx.TxIn {

			txInBlock := blockTxIdx[txIn.PreviousOutPoint.Hash]
			txInQueue := b.pendingBlocks.Tx(txIn.PreviousOutPoint.Hash)

			if txInBlock == nil && txInQueue == nil {
				missingIns = append(missingIns, storage.TxOutId{
					TxHash: txIn.PreviousOutPoint.Hash, 
					Nout:   txIn.PreviousOutPoint.Index,})
			}
		}
	}

	// Load missing transaction outputs from storage
	missingData, err := b.storageCache.BulkGetTxOut(missingIns)
	if err != nil {
		return nil, err
	}

	// Add missing inputs to transactions
	var missingIdx int = 0 // Index for the next unused missing input
	var output *primitives.TxOut

	for TxIdx := 0; TxIdx < len(block.Transactions); TxIdx += 1 {
	
		// If it is coinbase transaction it has only one input
		if blockchain.IsCoinBaseTx(block.Transactions[TxIdx]) {
			continue
		}

		for _, wireTxIn := range block.Transactions[TxIdx].TxIn {

			if tx := b.pendingBlocks.Tx(wireTxIn.PreviousOutPoint.Hash); tx != nil {
				// The input is an output from a pending block
				output = tx.Out[wireTxIn.PreviousOutPoint.Index]
			} else if tx := blockTxIdx[wireTxIn.PreviousOutPoint.Hash]; tx != nil {
				// The input is an output from the current block
				output = tx.Out[wireTxIn.PreviousOutPoint.Index]
			} else {
				// The input was retrieved from storage
				output = primitives.NewTxOut(
					&wireTxIn.PreviousOutPoint.Hash, 
					wireTxIn.PreviousOutPoint.Index,
					missingData[missingIdx].Addr,
					missingData[missingIdx].Value)
				missingIdx += 1
			}
			transactions[TxIdx].AddIn(output)
		}
	}

	pBlock := primitives.NewBlock(*bHash, block.Header.PrevBlock, height)
	pBlock.Transactions = transactions
	return pBlock, nil
}

// AddBlock adds a wire.Block to the manager returning primitives.Block equivalent
func (b *BlockManager) AddBlock(block *wire.MsgBlock, blockHash *chainhash.Hash) (*primitives.Block, error) {
	
	// Generate	block and add to pending
	pBlock, err := b.buildBlock(blockHash, block, uint64(b.height+1))
	if err != nil {
		return nil, err
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
	now := time.Now()
	elapsed := now.Sub(b.lastTime).Minutes() // time since last block
	if b.storageCache.UncommittedLen() > StorageCommitSize || elapsed > 2.0 {
		
		log.Print("Commit: ", b.storageCache.GetHeight())
		err := b.storageCache.Commit()
		if err != nil {
			return nil, err
		}
		b.lastTime = time.Now() // In case commit was too slow
	}
	b.lastTime = now

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


package crawler


import (
	"log"
	_ "sync"
	_ "time"
	"errors"

	"github.com/phf/go-queue/queue"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/crawler/storage"
)

const (
	UtxoCacheSize = 400000
	
	// Average number of transaction inputs in a block
	AverageBlockInputs = 2048 * 10 // Transactions * Inputs

	// Number of pending commits ve
	StorageCommitSize = 400000
)

var ErrBacktrackLimit = errors.New("not found")


type BlockManager struct {

	// Last added block height
	height int64

	// 
	storageCache *storage.StorageCache

	// Confirmations required for a block to be elegible for storage
	confirmations uint16

	// Blocks waiting for enough confirmations before committing to storage
	pendingBlocks *queue.Queue

	// Transactions provided by pendingBlocks
	pendingTx map[chainhash.Hash]*primitives.Tx
}

// NewBlockManager
func NewBlockManager(sto storage.Storage, cacheSize int, confirmations uint16) (*BlockManager, error) {

	cache, err := storage.NewStorageCache(sto, cacheSize)
	if err != nil {
		return nil, err
	}
	
	manager := BlockManager {
		height: 		cache.GetHeight(),
		storageCache:	cache,
		confirmations:	confirmations,
		pendingBlocks:	queue.New(),
		pendingTx:   make(map[chainhash.Hash]*primitives.Tx),
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
	transactionIdx := make(map[chainhash.Hash]*primitives.Tx, len(block.Transactions))
	for _, wireTx := range block.Transactions {
		tx := b.buildTx(wireTx)
		transactions = append(transactions, tx)
		transactionIdx[*tx.Hash] = tx
	}

	// Find all the transaction inputs that need to be retrieved from storage
	missingIns := make([]storage.TxOutId, 0, AverageBlockInputs)
	inputCount := 0 // Number of inputs spent by the block transactions
	
	for _, wireTx := range block.Transactions {
		for _, txIn := range wireTx.TxIn {

			_, ok1 := b.pendingTx[txIn.PreviousOutPoint.Hash]
			_, ok2 := transactionIdx[txIn.PreviousOutPoint.Hash]
			
			if !ok1 && !ok2 {
				missingIns = append(missingIns, storage.TxOutId{
					TxHash: txIn.PreviousOutPoint.Hash, 
					Nout:   txIn.PreviousOutPoint.Index,})
			}
			inputCount += 1
		}
	}

	// Load missing transactions from storage
	log.Print(len(missingIns))
	missingData, err := b.storageCache.BulkGetTxOut(missingIns)
	if err != nil {
		return nil, err
	}

	// Add missing inputs to transactions
	var missingIdx int = 0 // Index for the next unused missing input
	var output *primitives.TxOut

	for TxIdx := 0; TxIdx < len(block.Transactions); TxIdx += 1 {
		
		for _, wireTxIn := range block.Transactions[TxIdx].TxIn {

			if tx, ok := b.pendingTx[wireTxIn.PreviousOutPoint.Hash]; ok {
				// The input is an output from a pending block
				output = tx.Out[wireTxIn.PreviousOutPoint.Index]
			} else if tx, ok := transactionIdx[wireTxIn.PreviousOutPoint.Hash]; ok {
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


// storeBlock adds block outputs and removes blocks inputs to and from storage
func (b *BlockManager) storeBlock(block *primitives.Block) error {
	for _, tx := range block.Transactions {
	
		// Add transaction outputs to storage
		for _, out := range tx.Out {
			if out.Addr != "" && out.Value != 0 {
				b.storageCache.AddTxOut(*out)
			}
		}

		// Delete transaction inputs from storage
		for _, in := range tx.In {
			b.storageCache.DelTxOut(storage.TxOutId{TxHash: *in.TxHash, Nout: in.Nout})
		}
	}
	return nil
}

// AddBlock adds a wire.Block to the manager returning primitives.Block equivalent
func (b *BlockManager) AddBlock(block *wire.MsgBlock) (*primitives.Block, error) {

	blockHash := block.BlockHash()
	
	// Generate	block and add to pending
	pBlock, err := b.buildBlock(&blockHash, block, uint64(b.height))
	if err != nil {
		return nil, err
	}
	
	b.pendingBlocks.PushBack(pBlock)
	for _, tx := range pBlock.Transactions {
		b.pendingTx[*tx.Hash] = tx
	}
	
	// Update current height
	b.height += 1

	// Add confirmed block and height to storage cache
	if b.pendingBlocks.Len() > int(b.confirmations) {
		confirmedBlock := b.pendingBlocks.PopFront().(*primitives.Block)
		b.storeBlock(confirmedBlock)
		for _, tx := range confirmedBlock.Transactions {
			delete(b.pendingTx, *tx.Hash)
		}

		// Update height
		cacheHeight := b.storageCache.GetHeight()
		b.storageCache.SetHeight(cacheHeight+1)
	}

	// Commit cache when there are enough changes
	// TODO: or enough time has passed
	if b.storageCache.UncommittedLen() > StorageCommitSize {
		err := b.storageCache.Commit()
		if err != nil {
			return nil, err
		}
	}

	return pBlock, nil
}

// BacktrackBlock
func (b *BlockManager) BacktrackBlock() (*primitives.Block, error){
	if b.pendingBlocks.Len() == 0 {
		return nil, ErrBacktrackLimit
	}

	b.height -= 1

	block := b.pendingBlocks.PopBack().(*primitives.Block)
	for _, tx := range block.Transactions {
		delete(b.pendingTx, *tx.Hash)
	}
	return block, nil
}

// GetHeight returns the last block height
func (b *BlockManager) Height() int64 {
	return b.height
}




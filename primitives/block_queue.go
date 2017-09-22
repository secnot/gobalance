package primitives

import (
	"github.com/phf/go-queue/queue"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type BlockQueue struct {

	//
	blocks *queue.Queue
	
	// Balance delta for the queue blocks
	balance map[string]int64

	// Transaction index
	txIndex map[chainhash.Hash]*Tx
}

// NewBlockQueue 
func NewBlockQueue() *BlockQueue {

	return &BlockQueue {
		blocks:		queue.New(),
		balance:    make(map[string]int64),
		txIndex:    make(map[chainhash.Hash]*Tx),
	}
}

// updateBalance
func (b *BlockQueue) updateBalance(address string, balance int64) {
	
	new_balance := b.balance[address] + balance
	b.balance[address] = new_balance
	
	if new_balance == 0 {
		delete(b.balance, address)
	}
}

// blockUpdatBalance add or substract block outputs and inputs from balance 
func (b *BlockQueue)blockUpdateBalance(block *Block, reverse bool) {
	
	var sign int64 = 1
	if reverse {
		sign = -1
	}

	for _, tx := range block.Transactions {
	
		// Add outputs
		for _, out := range tx.Out {
			if out.Addr == "" || out.Value == 0 {
				continue
			}
			b.updateBalance(out.Addr, out.Value*sign)
		}

		// Substract inputs
		for _, in := range tx.In {
			if in.Addr == "" || in.Value == 0 {
				continue
			}
			b.updateBalance(in.Addr, -in.Value*sign)
		}
	}
}

// blockUpdateTxIndex 
func (b *BlockQueue)blockUpdateTxIndex(block *Block, reverse bool) {

	for _, tx := range block.Transactions {

		if reverse {
			delete(b.txIndex, *(tx.Hash))
		} else {
			b.txIndex[*tx.Hash] = tx
		}
	}
}


func (b *BlockQueue) PushBack(block *Block) {

	b.blocks.PushBack(block)

	// Update balance index
	b.blockUpdateBalance(block, false)
	
	// Update transaction index
	b.blockUpdateTxIndex(block, false)
}

func (b *BlockQueue) PushFront(block *Block) {
	
	b.blocks.PushFront(block)
	
	// Update balance index
	b.blockUpdateBalance(block, false)
	
	// Update transaction index
	b.blockUpdateTxIndex(block, false)
}


// PopLastBlock
func (b *BlockQueue) PopBack() *Block {
	
	block := b.blocks.PopBack()
	if block == nil {
		return nil
	}

	// Update balance index
	b.blockUpdateBalance(block.(*Block), true)
	
	// Update transaction index
	b.blockUpdateTxIndex(block.(*Block), true)

	return block.(*Block)
}

// PopFirstBlock
func (b *BlockQueue) PopFront() *Block {

	block := b.blocks.PopFront()
	if block == nil {
		return nil
	}

	// Update balance index
	b.blockUpdateBalance(block.(*Block), true)
	
	// Update transaction index
	b.blockUpdateTxIndex(block.(*Block), true)

	return block.(*Block)
}

// GetBalance returns the balance delta generated 
func (b *BlockQueue) GetBalance(address string) (balance int64, ok bool) {
	balance, ok = b.balance[address]
	return
}

// Len returns the number of blocks in the queue
func (b *BlockQueue) Len() int {
	return b.blocks.Len()
}

// TxCount returns the number of transactions in the queued blocks
func (b *BlockQueue) TxCount() int {
	return len(b.txIndex)
}

// ConstainsTx returns true if one block in the queue contains the transaction
func (b *BlockQueue) Tx(hash chainhash.Hash) *Tx {
	tx, ok := b.txIndex[hash]
	if ok {
		return tx
	} 
	return nil
}


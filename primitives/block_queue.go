package primitives

import (
	"github.com/secnot/gobalance/primitives/queue"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)


// txBlock store transactions and the block that contain them together
type txBlock struct{
	// Pointer to the transaction
	tx *Tx

	// Block that contains the transaction
	block *Block
}

type BlockQueue struct {

	//
	blocks *queue.Queue
	
	// Balance delta for the queue blocks
	addrBalance map[string]int64

	// Index addresses to the list of transactions that contain them 
	addrTx map[string]*queue.Queue

	// Transaction index
	txIndex map[chainhash.Hash]txBlock
}

// NewBlockQueue 
func NewBlockQueue() *BlockQueue {
	
	return &BlockQueue {
		blocks:		 queue.New(),
		addrBalance: make(map[string]int64),
		addrTx:      make(map[string]*queue.Queue),
		txIndex:     make(map[chainhash.Hash]txBlock),
	}
}

// addAddress adds tx to both address indexes
func (b *BlockQueue) addAddress(address string, balance int64, tx *Tx) {
	
	// Add tx to address index
	if b.addrTx[address] == nil {
		b.addrTx[address] = queue.New()
	}

	b.addrTx[address] .PushBack(tx)

	// Update accumulated address balance
	new_balance := b.addrBalance[address] + balance
	if new_balance == 0 {
		delete(b.addrBalance, address)
	} else {
		b.addrBalance[address] = new_balance
	}
}

// delAddress removes tx from both address indexes
func (b *BlockQueue) delAddress(address string, balance int64, tx *Tx) {

	// Remove tx from address index
	q := b.addrTx[address]
	q.PopFront()
	
	if q.Len() == 0 {
		delete(b.addrTx, address)
	}

	// Update accumulated address balance
	new_balance := b.addrBalance[address] - balance
	if new_balance == 0 {
		delete(b.addrBalance, address)
	} else {
		b.addrBalance[address] = new_balance
	}
}


// blockUpdatBalance add or substract block outputs and inputs from balance 
func (b *BlockQueue)blockUpdateAddress(block *Block, reverse bool) {
	
	for _, tx := range block.Transactions {
		if reverse {
			tx.ForEachAddress(b.delAddress)
		} else {
			tx.ForEachAddress(b.addAddress)
		}
	}
}

// blockUpdateTxIndex 
func (b *BlockQueue)blockUpdateTxIndex(block *Block, reverse bool) {

	for _, tx := range block.Transactions {

		if reverse {
			delete(b.txIndex, *(tx.Hash))
		} else {
			b.txIndex[*tx.Hash] = txBlock{tx: tx, block: block}
		}
	}
}

// PushBack inserts a new block at the back of the queue
func (b *BlockQueue) PushBack(block *Block) {

	b.blocks.PushBack(block)

	// Update balance index
	b.blockUpdateAddress(block, false)
	
	// Update transaction index
	b.blockUpdateTxIndex(block, false)
}

// PushFront inserts a new block at the front of the queue
func (b *BlockQueue) PushFront(block *Block) {
	
	b.blocks.PushFront(block)
	
	// Update balance index
	b.blockUpdateAddress(block, false)
	
	// Update transaction index
	b.blockUpdateTxIndex(block, false)
}


// PopBack removes and returns the last block of the queue or nil
func (b *BlockQueue) PopBack() *Block {
	
	block := b.blocks.PopBack()
	if block == nil {
		return nil
	}

	// Update address index
	b.blockUpdateAddress(block.(*Block), true)
	
	// Update transaction index
	b.blockUpdateTxIndex(block.(*Block), true)

	return block.(*Block)
}

// PopFirst removes and returns the first block of the queue or nil
func (b *BlockQueue) PopFront() *Block {

	block := b.blocks.PopFront()
	if block == nil {
		return nil
	}

	// Update address index
	b.blockUpdateAddress(block.(*Block), true)
	
	// Update transaction index
	b.blockUpdateTxIndex(block.(*Block), true)

	return block.(*Block)
}

// Front returns a pointer to the block at the front of the queue without altering
// the queue
func (b *BlockQueue) Front() *Block{
	block := b.blocks.Back()
	if block == nil {
		return nil 
	}
	return block.(*Block)
}

// Back return a pointer to the block at the back of the queue without altering
// the queue
func (b *BlockQueue) Back() *Block{
	block := b.blocks.Back()
	if block == nil {
		return nil 
	} 
	return block.(*Block)
}

// GetBalance returns the balance delta generated 
func (b *BlockQueue) GetBalance(address string) (balance int64, ok bool) {
	balance, ok = b.addrBalance[address]
	return
}

// GetTx returns all the transactions where the address took part
func (b *BlockQueue) GetTx(address string) []*Tx {
	
	transactionQ, ok := b.addrTx[address]
	if !ok {
		return nil
	}
	
	transactions := make([]*Tx, 0, transactionQ.Len())
	iter := transactionQ.Iter()
	for tx, finished := iter.Next(); !finished; tx, finished = iter.Next() {
		transactions = append(transactions, tx.(* Tx))
	}

	return transactions
}

// Len returns the number of blocks in the queue
func (b *BlockQueue) Len() int {
	return b.blocks.Len()
}

// TxCount returns the number of transactions in the queued blocks
func (b *BlockQueue) TxCount() int {
	return len(b.txIndex)
}

// Tx returns the transaction and the block containing it.
func (b *BlockQueue) Tx(hash chainhash.Hash) (*Tx, *Block) {
	tx, ok := b.txIndex[hash]
	if ok {
		return tx.tx, tx.block
	} 
	return nil, nil
}


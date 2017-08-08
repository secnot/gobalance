package balance

import (
	"log"
	"sync"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/balance/storage"
	"github.com/secnot/gobalance/crawler"
	"github.com/phf/go-queue/queue"
)

const (
	BalanceCacheSize = 100000

	// Number of pending updates required to trigger a Commit
	BalanceCommitSize = 400000
)



type BalanceProcessor struct {

	// Concurrency lock
	sync.RWMutex

	// A cache for the storage 
	balance *storage.StorageProxyCache

	// Height of the last processed processed
	height int64

	// Accumulated balance updates for the blocks without enough confirmations 
	// to be stored (by address)
	pending map[string]int64

	// Block height queue waitting go be 
	blockQueue *queue.Queue
}


// NewBalanceProcessor
func NewBalanceProcessor(store storage.Storage, caseSize int) (balance *BalanceProcessor) {
	storageCache, err := storage.NewStorageProxyCache(store, BalanceCacheSize)
	if err != nil {
		log.Print(err)
		return nil
	}

	return &BalanceProcessor{
		balance: storageCache,
		height: storageCache.Height(),
		blockQueue: queue.New(),
		pending: make(map[string]int64),
	}
}

// GetBalance returns the address balance or 0
func (b *BalanceProcessor) GetBalance(address string) (bal int64, err error){
	b.RLock()
	bal, err = b.balance.Get(address)
	bal += b.pending[address]
	b.RUnlock()
	return 
}

// addToPending adds block inputs and outputs to pending map
func (b *BalanceProcessor) addToPending(block *primitives.Block) {
	for _, tx := range block.Transactions {
		// Add Outpus
		for _, out := range tx.Out {
			if out.Addr != "" && out.Value != 0 {
				b.pending[out.Addr] += out.Value
				
				if b.pending[out.Addr] == 0 {
					delete(b.pending, out.Addr)
				}
			}
		}

		// Substract inputs
		for _, in := range tx.In {
			if in.Addr != "" && in.Value != 0 {
				b.pending[in.Addr] -= in.Value
				
				if b.pending[in.Addr] == 0 {
					delete(b.pending, in.Addr)
				}
			}
		}
	}
}

// delFromPending deletes block input and outputs from pending map
func (b *BalanceProcessor) delFromPending(block *primitives.Block) {
	for _, tx := range block.Transactions {
		// Add Outpus
		for _, out := range tx.Out {
			if out.Addr != "" && out.Value != 0 {
				b.pending[out.Addr] -= out.Value
				
				if b.pending[out.Addr] == 0 {
					delete(b.pending, out.Addr)
				}
			}
		}

		// Substract inputs
		for _, in := range tx.In {
			if in.Addr != "" && in.Value != 0 {
				b.pending[in.Addr] += in.Value
				
				if b.pending[in.Addr] == 0 {
					delete(b.pending, in.Addr)
				}
			}
		}
	}
}

// addBlock enques a new block
func (b *BalanceProcessor) addBlock(block *primitives.Block) {

	 b.Lock()
	 b.addToPending(block)
	 b.blockQueue.PushBack(block)
	 b.height = int64(block.Height)
	 b.Unlock()
}

// storeOldestBlock dequeues and stores the oldest block
func (b *BalanceProcessor) storeOldestBlock() {

	b.Lock()
	iblock := b.blockQueue.PopFront()
	if iblock == nil {
		b.Unlock()
		return
	}
	block := iblock.(*primitives.Block)
	b.delFromPending(block)

	// Commit to storage
	for _, tx := range block.Transactions {		
		
		// Add Outpus
		for _, out := range tx.Out {
			if out.Addr != "" {
				b.balance.Update(out.Addr, out.Value)
			}
		}

		// Substract inputs
		for _, in := range tx.In {
			if in.Addr != "" {
				b.balance.Update(in.Addr, -in.Value)
			}
		}
	}
	b.balance.SetHeight(int64(block.Height))
	b.Unlock()
}

// NewBlock is called by the crawler to send the latest block 
func (b *BalanceProcessor) NewBlock(block *primitives.Block) {
	
	// Ignore blocks older than the current height, the crawler is catching up
	if int64(block.Height) <= b.height {
		return 
	}

	// Check no blocks were skiped
	if int64(block.Height) != b.height+1 {
		log.Panicf("BalanceProcessor: Height is %v but the new block height is %v",
			b.height, block.Height)
	}

	// Store new block and update pending
	b.addBlock(block)

	// Pop oldest block to send to storage, if it has enough confirmations
	if b.blockQueue.Len() < crawler.BacktrackLogSize {
		return
	}
	b.storeOldestBlock()

	// Commit only if there are enough pending updates in storage
	// TODO: Don't accumulate updates if more than 5 minutes have passed
	// since the last (ie .- initial sync has finished)
	if b.balance.UncommittedLen() > BalanceCommitSize {
		err := b.balance.Commit() 
		if err != nil {
			log.Panic("proxy.Commit(): ", err)
		}
	}
	return
}

// BacktrackBlock is called by the crawler to invalidate the last block
func (b *BalanceProcessor) BacktrackBlock(block *primitives.Block) {
	bblock := b.blockQueue.PopBack()
	if bblock == nil {
		log.Panic("BalanceProcessor: Reached backtrack limit.")
	}

	if bblock.(*primitives.Block).Height != block.Height {
		log.Panic("BalanceProcessor: Backtrack block height missmatch")
	}

	b.delFromPending(bblock.(*primitives.Block))	
}

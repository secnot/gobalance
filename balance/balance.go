package balance

import (
	"sync"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/balance/storage"
	"github.com/secnot/simplelru
)


type BalanceProcessor struct {

	// Concurrency lock
	sync.RWMutex

	// Updates not yet committed to storage
	pending map[string]int64

	// Some kind of persistent storage
	storage Storage

	// A cache for the storage 
	cache simplelru.LRUCache

}

func NewBalanceProcessor() (balance *BalanceProcessor) {
	return nil
}




type Balance struct {

	// Only blocked
	sync.RWMutex

	// Current block height
	Height uint64 

}


func (b *Balance) AddBlock() {
}


// Backtrack a single block
func (b *Balance) Backtrack() (err error) {
	return nil
}
	
// NewBlock is called by the crawler to send the latest discovered block 
func (b *BalanceProcessor) NewBlock(block *primitives.Block) {
	return
}

// BacktrackBlock is called by the crawler to invalidate the last valid block
func (b *BalanceProcessor) BacktrackBlock(block *primitives.Block) {
	return
}

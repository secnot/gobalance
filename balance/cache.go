package balance


import (
	"github.com/secnot/simplelru"
	"github.com/secnot/gobalance/block_manager"
	"github.com/secnot/gobalance/primitives"
)




type BalanceCache struct {
	size int
	manager *block_manager.BlockManager
	cache *simplelru.LRUCache
}

func NewBalanceCache(size int, manager *block_manager.BlockManager) *BalanceCache {
	return &BalanceCache{
		size:    size,
		manager: manager,
		cache:   simplelru.NewLRUCache(size, 1000),
	}
}


//
func (b *BalanceCache) addBalance(addr string, balance int64, tx *primitives.Tx) {
	if currentBalance, ok := b.cache.Get(addr); ok {
		b.cache.Set(addr, currentBalance.(int64)+balance)
	}
}

//
func (b *BalanceCache) remBalance(addr string, balance int64, tx *primitives.Tx) {
	if currentBalance, ok := b.cache.Get(addr); ok {
		b.cache.Set(addr, currentBalance.(int64)-balance)
	}
}


// NewBlock adds a new block to cache
func (b *BalanceCache) NewBlock(block *primitives.Block) {

	for _, tx := range block.Transactions {
		tx.ForEachAddress(b.addBalance)
	}
}

// Bactrack a block from cache
func (b *BalanceCache) Backtrack(block *primitives.Block) {
	for _, tx := range block.Transactions {
		tx.ForEachAddress(b.remBalance)
	}
}

func (b *BalanceCache) GetBalance(address string) int64 {
	
	// Check cache fro balance
	if balance, ok := b.cache.Get(address); ok{
		return balance.(int64)
	}

	// If there was a cache miss retrieve balance from block_manager
	balance := b.manager.GetBalance(address)

	b.cache.Set(address, balance)
	return balance
}



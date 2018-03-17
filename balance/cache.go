package balance


import (
	"github.com/secnot/simplelru"
	"github.com/secnot/gobalance/block_manager"
	"github.com/secnot/gobalance/primitives"
)




type Cache struct {
	size int
	manager *block_manager.BlockManager
	cache *simplelru.LRUCache
}

//
func NewCache(size int, manager *block_manager.BlockManager) *Cache {
	return &Cache{
		size:    size,
		manager: manager,
		cache:   simplelru.NewLRUCache(size, 1000),
	}
}

//
func (c *Cache) addBalance(addr string, balance int64, tx *primitives.Tx) {
	if currentBalance, ok := c.cache.Get(addr); ok {
		c.cache.Set(addr, currentBalance.(int64)+balance)
	}
}

//
func (c *Cache) remBalance(addr string, balance int64, tx *primitives.Tx) {
	if currentBalance, ok := c.cache.Get(addr); ok {
		c.cache.Set(addr, currentBalance.(int64)-balance)
	}
}


// NewBlock adds a new block to cache
func (c *Cache) NewBlock(block *primitives.Block) {

	for _, tx := range block.Transactions {
		tx.ForEachAddress(c.addBalance)
	}
}

// Bactrack a block from cache
func (c *Cache) Backtrack(block *primitives.Block) {
	for _, tx := range block.Transactions {
		tx.ForEachAddress(c.remBalance)
	}
}

func (c *Cache) GetBalance(address string) int64 {
	
	// Check cache fro balance
	if balance, ok := c.cache.Get(address); ok{
		return balance.(int64)
	}

	// If there was a cache miss retrieve balance from block_manager
	balance := c.manager.GetBalance(address)

	c.cache.Set(address, balance)
	return balance
}



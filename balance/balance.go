package balance


import (
	"github.com/secnot/simplelru"
	"github.com/secnot/gobalance/crawler"
	"github.com/secnot/gobalance/primitives"
)



var ( 
	BalanceRequestChan   = make(chan BalanceRequest, 100)
)

type BalanceRequest struct {
	Address string
	Response chan int64
}



type BalanceCache struct {
	size int

	cache *simplelru.LRUCache
}

func NewBalanceCache(size int) *BalanceCache {
	return &BalanceCache{
		size: size,
		cache: simplelru.NewLRUCache(size, 1000),
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

	// If there was a cache miss retrieve balance from crawler
	balance := crawler.GetBalance(address)

	b.cache.Set(address, balance)
	return balance
}


func BalanceRoutine(cacheSize int) {
	cache := NewBalanceCache(cacheSize)

	updateChan := crawler.Subscribe(10)
	
	for {

		select {
		case update := <- updateChan:			
		
			switch update.Class {
			case crawler.OP_NEWBLOCK:
				cache.NewBlock(update.Block)
			case crawler.OP_BACKTRACK:
				cache.Backtrack(update.Block)
			}

		case request := <- BalanceRequestChan:
			request.Response <- cache.GetBalance(request.Address)
		}
		
	}
}

func GetBalance(address string) int64 {
	
	responseChan := make(chan int64)
	BalanceRequestChan <- BalanceRequest{Address: address, Response: responseChan}
	balance := <- responseChan
	close(responseChan)
	return balance
}

package balance

import (
	"log"
	"sync"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/balance/storage"
	"github.com/secnot/simplelru"
	"github.com/phf/go-queue/queue"
)

const (
	BalanceCacheSize = 200000
	BalanceConcurrentQueryWorkers = 10
)

type Balance struct {

	// 
	sync.RWMutex
	
	// Balance cache, this are cached from storage
	cache *simplelru.LRUCache

	// Persistent storage
	store storage.Storage

	// Current height
	height int64

	// Updates
	updates map[string]int64
}

func NewBalance(store storage.Storage) (*Balance, error) {
	
	height, err := store.GetHeight()
	if err != nil {
		return nil, err
	}

	fetchBalance := func (address interface{}) (value interface{}, ok bool) {
	
		balance, err := store.Get(address.(string))
		if err != nil {
			return nil, false
		}

		return balance, true
	}

	cache := simplelru.NewFetchingLRUCache(
		BalanceCacheSize,
		BalanceCacheSize/100+1,
		fetchBalance,
		BalanceConcurrentQueryWorkers,
		BalanceConcurrentQueryWorkers*3)
	
	return &Balance{
		cache:   cache,
		store:   store,
		height:  height,
		updates: make(map[string]int64),
	}, nil
}

func (b *Balance) GetBalance(address string) (int64, error) {
	b.RLock()
	defer b.RUnlock()
	balance, ok := b.cache.Get(address)
	if !ok {
		// Storage failed if the address didn't existe it should have
		// returned 0, true
		log.Panicf("Storage failure: %v", b.store)
	}

	return balance.(int64)+b.updates[address], nil
}

func (b *Balance) UpdateBalance(address string, value int64) {
	b.Lock()
	current := b.updates[address]
	if current+value == 0{	
		delete(b.updates, address)
	} else {
		b.updates[address] = current+value
	}
	b.Unlock()
}

// Len returns the number of pending updates
func (b *Balance) Len() int {
	return len(b.updates)
}

// Commit updates to storage
// TODO: Reuse missing/insert/update/remove arrays between calls
func (b *Balance) Commit(height int64) error {
	b.Lock()
	defer b.Unlock()
	
	// Load into cache the balance for all pending updates.
	var updatedAddr = make([]string, len(b.updates))
	
	i := 0
	for addr, _ := range b.updates {
		updatedAddr[i] = addr
		i++
	}

	balances, err := b.store.BulkGet(updatedAddr)
	if err != nil {
		return err
	}

	// Split addresses into Insert/Update/Remove slices
	var insert = make([]storage.AddressBalancePair, len(updatedAddr))
	var update = make([]storage.AddressBalancePair, len(updatedAddr))
	var remove = make([]string, len(updatedAddr))

	for n, addr := range updatedAddr {
		balance := balances[n]
		upValue := b.updates[addr]
		if balance == 0 {
			// Insert
			insert = append(insert, storage.AddressBalancePair{addr, upValue})
		} else if balance + upValue == 0 {
			// Delete
			remove = append(remove, addr)
		} else {
			// Update
			update = append(update, storage.AddressBalancePair{addr, balance+upValue})
		}

	}

	err = b.store.BulkUpdate(insert, update, remove, height)
	if err != nil {
		return err
	}
	b.height = height

	// Update cache and cleanup Stats
	cachedCount := 0
	for n, addr := range updatedAddr {
		balance := balances[n]
		upValue := b.updates[addr]
		if b.cache.Contains(addr) {
			cachedCount++
		}
		b.cache.Set(addr, balance+upValue)
	}
	log.Print(height, cachedCount, len(updatedAddr))
	b.updates = make(map[string]int64)
	return nil
}




type BalanceProcessor struct {

	// Concurrency lock
	sync.RWMutex

	// Updates not yet committed to storage
	//pending map[string]int64

	// A cache for the storage 
	//cache *simplelru.LRUCache
	balance *Balance

	// Block queue waitting go be 
	blockQueue *queue.Queue
}


// NewBalanceProcessor
func NewBalanceProcessor(store storage.Storage, caseSize int) (balance *BalanceProcessor) {
	bal, err := NewBalance(store)
	if err != nil {
		return nil
	}
	
	return &BalanceProcessor{
		balance: bal,
		blockQueue: queue.New(),
	}
}

func (b *BalanceProcessor) GetBalance(address string) {
	return
}

// NewBlock is called by the crawler to send the latest block 
func (b *BalanceProcessor) NewBlock(block *primitives.Block) {
	for _, tx := range block.Transactions {
		
		// Add Outpus to address balance
		for _, out := range tx.Out {
			if out.Addr != "" {
				b.balance.UpdateBalance(out.Addr, out.Value)
			}
		}

		// Substract inputs from address balance
		for _, in := range tx.In {
			if in.Addr != "" {
				b.balance.UpdateBalance(in.Addr, -in.Value)
			}
		}
	}

	if block.Height%5 == 0 {
		b.balance.Commit(int64(block.Height))
	}
	return
}

// BacktrackBlock is called by the crawler to invalidate the last block
func (b *BalanceProcessor) BacktrackBlock(block *primitives.Block) {
	// TODO:
	return
}

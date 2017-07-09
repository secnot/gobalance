package balance

import (
	"sync"
	"fmt"
	"github.com/secnot/simplelru"
	_ "github.com/secnot/orderedmap"
)


type StorageProxyCache struct {

	sync.RWMutex

	// Storage cache
	cache *simplelru.LRUCache

	// Max cache size
	cache_size int

	// Updates received but not yet commited
	pending map[string]int64

	// Storage being proxied
	storage Storage

	// This is the proxy height, not the stored height
	height int64
}


//
func NewStorageProxyCache(storage Storage, cache_size int) (s *StorageProxyCache) {
	
	// Create a new 
	fetchFunc := func (address interface{}) (balance interface{}, ok bool) {
		balance, err := storage.Get(address.(string))
		if err != nil {
			// Some storage error
			return 0, false
		}
		return balance, true
	}

	height, err := storage.GetHeight()
	if err != nil {
		return nil
	}

	cache := simplelru.NewFetchingLRUCache(
		cache_size,
		cache_size/100+1, // pruneSize
		fetchFunc,
		1,		// Workers
		1000)	// JobQueueSize

	proxy := &StorageProxyCache {
		cache: cache,
		pending: make(map[string]int64),
		cache_size: cache_size,
		storage: storage,
		height: height,
	}

	return proxy
}

// GetHeight
func (s *StorageProxyCache) GetHeight() (height int64, err error) {
	s.RLock()
	height, err = s.height, nil
	s.RUnlock()
	return
}

// SetHeight
func (s *StorageProxyCache) SetHeight(height int64) {
	s.Lock()
	s.height = height
	s.Unlock()
	return
}

// Get address balance
func (s *StorageProxyCache) Get(address string) (balance int64, err error) {

	s.RLock()
	defer s.RUnlock()
	storedBalance, ok := s.cache.Get(address) 
	if !ok {
		//There was an error while fetching the balance from storage
		return 0, NewStorageError("Unable to access storage")
	}
		
	// Add pending updates and return...
	balance = storedBalance.(int64) + s.pending[address]
	if balance < 0 {
		err = NewNegativeBalanceError(fmt.Sprintf("%v balance is %v", address, balance))
	} else {
		err = nil
	}

	return
}

// Update address balance (doesn't commit changes to storage)
func (s *StorageProxyCache) Update(address string, amount int64) {
	s.Lock()
	if amount != 0 {
		s.pending[address] += amount
		if s.pending[address] == 0 {
			delete(s.pending, address)
		}
	}
	s.Unlock()
}

// Commit pending updates to storage
func (s *StorageProxyCache) Commit() (err error){

	s.Lock()


	// Find all pending updates whose balance is not cached
	var missing []string
	for address, _ := range s.pending {
		if !s.cache.Contains(address) {
			missing = append(missing, address)
		}
	}
	
	missingBalance, err := s.storage.BulkGet(missing)
	if err != nil {
		s.Unlock()
		return err
	}

	// Add enough space to cache for all the missing balances
	s.cache.Resize(s.cache_size+len(missing), s.cache_size/100+1)
	for n, address := range missing {
		s.cache.Set(address, missingBalance[n])
	}

	// Split pending into updates/inserts/deletions
	var update []AddressBalancePair
	var insert []AddressBalancePair
	var remove []string
	for address, amount := range s.pending {
		ibalance, _ := s.cache.Get(address) // All should be cached
		balance := ibalance.(int64)
		if balance + amount < 0 {
			errMsg := fmt.Sprintf("Commit(): \"%v\" balance is negative (%v)", 
				address, balance+amount)
			err = NewNegativeBalanceError(errMsg)
			break
		}
		
		if balance == 0 {
			// INSERT
			insert = append(insert, AddressBalancePair{address, amount})
		} else if balance + amount == 0 {
			// DELETE
			remove = append(remove, address)
		} else {
			// UPDATE
			update = append(update, AddressBalancePair{address, balance+amount})
		}
	}

	// Return cache to its original size
	s.cache.Resize(s.cache_size, s.cache_size/100+1)
	//TODO: Remove from cache missing address added instead of the oldest
	if err != nil { // Negative balance error
		s.Unlock()
		return err
	}
	
	// Update storage
	err = s.storage.BulkUpdate(insert, update, remove, s.height)
	if err == nil {
		// Update cached balances
		for address, update := range s.pending {
			if balance, ok := s.cache.Peek(address); ok {
				s.cache.Set(address, balance.(int64) + update)
			}
		}

		// Clear pending updates
		s.pending = make(map[string]int64)
	}

	s.Unlock()
	return err
}

// UncommitedLen returns the number of updates not yet commited
func (s *StorageProxyCache) UncommittedLen() (length int) {
	s.RLock()
	length = len(s.pending)
	s.RUnlock()
	return
}

// Len returns the number or stored balances
func (s *StorageProxyCache) Len() (length int, err error) {
	s.RLock()
	length, err = s.storage.Len()
	s.RUnlock()
	return
}



// Clear balance cache
func (s *StorageProxyCache) CacheClear() {
	s.Lock()
	s.cache.Purge()
	s.Unlock()
}

// Elements stored by balance cache
func (s *StorageProxyCache) CacheLen() (length int){
	s.RLock()
	length = s.cache.Len()
	s.RUnlock()
	return
}


// GetCacheStats return hit/miss count for balance cache
func (s *StorageProxyCache) GetStats() (hit uint64, miss uint64) {
	s.RLock()
	hit, miss = s.cache.Stats()
	s.RUnlock()
	return
}

// ResetStats initialize stats to 0 hits / 0 miss 
func (s *StorageProxyCache) ResetStats() {
	s.Lock()
	s.cache.ResetStats()
	s.Unlock()
}



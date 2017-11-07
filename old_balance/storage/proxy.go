package storage

import (
	"sync"
	"fmt"
	"github.com/secnot/simplelru"
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
func NewStorageProxyCache(storage Storage, cache_size int) (s *StorageProxyCache, err error) {
	
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
		return nil, err
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

	return proxy, nil
}

// GetHeight
func (s *StorageProxyCache) Height() (height int64) {
	s.RLock()
	height = s.height
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

	// Find all not cached pending updates
	updates := make([]AddressBalancePair, len(s.pending))
	i := 0
	for address, amount := range s.pending {
		updates[i] = AddressBalancePair{address, amount}
		i++
	}
	
	// Update storage
	err = s.storage.BulkUpdate(updates, s.height)
	if err == nil {
		// Update balance cache
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



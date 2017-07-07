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

	// TODO: Initialize height from storage
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
		height: 0,
	}

	return proxy
}

// GetHeight
func (s *StorageProxyCache) GetHeight() (height uint64, err error) {
	s.RLock()
	height, err = s.height, nil
	s.RUnlock()
	return
}

// SetHeight
func (s *StorageProxyCache) SetHeight(height uint64) {
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
		return 0, NewErrorStorage("Unable to access storage")
	}
		
	// Add pending updates and return...
	balance = storedBalance.(int64) + s.pending[address]
	if balance < 0 {
		err = NewErrorNegativeBalance(fmt.Sprintf("%v balance is %v", address, balance))
	} else {
		err = nil
	}

	return
}

// Update address balance (doesn't commit changes to storage)
func (s *StorageProxyCache) Update(address string, ammount int64) {
	s.Lock()
	if ammount != 0 {
		s.pending[address] += ammount
		if s.pending[address] == {
			delete(s.pending, address)
		}
	}
	s.Unlock()
}

// Commit pending updates to storage
func (s *StorageProxyCache) Commit() (err error){

	var err error = nil
	s.Lock()


	// TODO: Load all missig 
	var misses []string // Address not cached that need to be loaded from storage
	for address, update := range s.pending() {
		balance, ok := s.cache.Peek(address)
		if !ok {
			// address isn't cached
			misses.append(address)
			continue
		}
	}
	
	queried, err := storage.BulkGet(misses)
	if err != nil {
		s.Unlock()
		return err
	}

	// Add enough space to cache for all the missing balances
	s.cache.Resize(s.cache_size+len(misses), s.cache_size/100+1)
	for kbpair := range queried {
		s.cache.Set(kbpair.address, kbpair.balance)
	}

	// Split pending updates into updates/inserts/deletions
	var update []AddressBalancePair
	var insert []AddressBalancePair
	var remove []string
	for address, update := range s.pending {
		balance, _ := s.cache.Get(address) // All pending balance should be cached
		if balance + update < 0 {
			errMsg := fmt.Sprintf("%v balance is %v", address, balance+update)
			err = NewErrorNegativeBalance(errorMsg)
			break
		}
		
		if balance == 0 {
			// INSERT
			insert.append(AddressBalancePair{address, update})
		} else if balance + update == 0 {
			// DELETE
			remove.append(address)
		} else {
			// UPDATE
			update.append(AddressBalancePair{address, balance+update})
		}
	}

	// Return cache to its original size
	s.cache.Resize(s.cache_size, s.cache_size/100+1)
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

	c.Unlock()
	return err
}


/*
// Clear balance cache
func (s *StorageProxyCache) ClearCache() {
}


// Return cached address balance or fail, also fail if the balance is a negative number
func (s *StorageProxyCache) cachedBalance(address string) (balance uint64, err error){
	// No lock here
	
}

//
func (s *StorageProxyCache) UncommitedLen() {
}

// 
func (s *StorageProxyCache) Len() {
}
*/

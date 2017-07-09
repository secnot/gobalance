package balance

import (
	"fmt"
	"testing"
)


// Initialize storage with n consecutive addresses 
// created address:balance -> address_0:0 .... address_n:n
// but because address_0 balance is 0 it shouldn't be stored
func storageInit(storage Storage, n int, height int64) {
	for i := int64(0); i < int64(n); i++ {
		address := fmt.Sprintf("address_%v", i)
		storage.Set(address, i)
	}
	storage.SetHeight(height)
}

func storageHasBalance(t *testing.T, storage Storage, address string, balance int64) {
	if retBalance, err := storage.Get(address); err != nil {
		errMsg := fmt.Sprintf("Get(%v): %v", address, err)
		t.Error(errMsg)
	} else if retBalance != balance {
		errMsg := fmt.Sprintf("Get(%v) returned %v expecting %v", 
							address, retBalance, balance)
		t.Error(errMsg)
	}

}

func storageHasLen(t *testing.T, storage Storage, size int) {
	if retSize, err := storage.Len(); err != nil {
		t.Error("Len():", err)
	} else if retSize != size {
		errMsg := fmt.Sprintf("Len() returned %v expecting %v", retSize, size)
		t.Error(errMsg)
	}
}

func cacheHasBalance(t *testing.T, cache *StorageProxyCache, address string, balance int64) {
	if retBalance, err := cache.Get(address); err != nil {
		errMsg := fmt.Sprintf("Get(%v): %v", address, err)
		t.Error(errMsg)
	} else if retBalance != balance {
		errMsg := fmt.Sprintf("Get(%v) returned %v expecting %v", 
							address, retBalance, balance)
		t.Error(errMsg)
	}
}

func cacheHasLen(t *testing.T, cache *StorageProxyCache, length int) {
	if l, err := cache.Len(); err != nil {
		errMsg := fmt.Sprintf("Len(): %v", err)
		t.Error(errMsg)
	} else if l != length{
		errMsg := fmt.Sprintf("Len(): returned %v expecting %v", 
							l, length)
		t.Error(errMsg)
	}
}

func cacheHasUncommittedLen(t *testing.T, cache *StorageProxyCache, length int) {
	if l := cache.UncommittedLen(); l != length {
		errMsg := fmt.Sprintf("Get(%v) returned %v expecting %v", 
							l, length)
		t.Error(errMsg)
	}
}

func cacheHasHeight(t *testing.T, cache *StorageProxyCache, height int64) {	
	if h, err := cache.GetHeight(); err != nil {
		errMsg := fmt.Sprintf("GetHeight(): %v", err)
		t.Error(errMsg)
	} else if h != height {
		errMsg := fmt.Sprintf("GetHeight() returned %v expecting %v", 
							h, height)
		t.Error(errMsg)
	}
}

func cacheHasStats(t *testing.T, cache *StorageProxyCache, hit uint64, miss uint64) {
	if h, m := cache.GetStats(); h != hit || m != miss {
		errMsg := fmt.Sprintf("GetStats() returned %v, %v expecting %v, %v", 
							h, m, hit, miss)
		t.Error(errMsg)
	}
}

// Verify cache size
func cacheHasSize(t *testing.T, cache *StorageProxyCache, cacheSize int) {

	// Store known values that later will be used to fill the cache
	for i := int64(0); i < int64(cacheSize*2); i++ {
		address := fmt.Sprintf("test_size_address_%v", i)
		cache.Update(address, i+1000)
	}
	cache.Commit()

	// Fill cache until there's a prune
	cache.CacheClear()
	cache.ResetStats()
	if cache.CacheLen() != 0 {
		t.Error("Cache should be empty after calling CacheClear")
	}

	var maxSize int = 0
	for i := int64(0); i < int64(cacheSize*2); i++ {
		address := fmt.Sprintf("test_size_address_%v", i)
		cache.Get(address)
		if cache.CacheLen() > maxSize {
			maxSize = cache.CacheLen()
		}
	}

	if maxSize > cacheSize {
		t.Error("The cache is larger than it should", maxSize, "vs", cacheSize)
	} else if maxSize < cacheSize {
		t.Error("The cache is smaller than it should", maxSize, "vs", cacheSize)
	}
}


// Test StorageProxy creation with intitialized storage
func TestNewStorageProxyInit(t *testing.T) {
	storage := NewMemoryStorage()
	storageInit(storage, 1000, 12) // Preload storage
	cache := NewStorageProxyCache(storage, 100)

	// Check there are 0 uncommited updates after creation
	cacheHasUncommittedLen(t, cache, 0)

	// Check intial storage length
	cacheHasLen(t, cache, 999)

	// Check initial stats
	cacheHasStats(t, cache, 0, 0)

	// Check proxy loads height from storage
	cacheHasHeight(t, cache, 12)

	// Check stored balance
	for i:= int64(0); i < int64(1000); i++ {
		address := fmt.Sprintf("address_%v", i)
		cacheHasBalance(t, cache, address, i)
	}

	cacheHasBalance(t, cache, "address_0", 0)
	cacheHasBalance(t, cache, "address_1000", 0)
}

// Test NewStorageProxy with uninitialized storage
func TestNewStorageProxy(t *testing.T) {
	storage := NewMemoryStorage()
	cache := NewStorageProxyCache(storage, 10000)
	
	// Check there are 0 uncommited updates after creation
	cacheHasUncommittedLen(t, cache, 0)

	// Check intial storage length
	cacheHasLen(t, cache, 0)
	
	// Check initial stats
	cacheHasStats(t, cache, 0, 0)

	// Check proxy loads height from storage
	cacheHasHeight(t, cache, -1)

	// Check stored balance
	cacheHasBalance(t, cache, "address_0", 0)
	cacheHasBalance(t, cache, "address_1", 0)
}


// Test update
func TestStorageProxyUpdate(t *testing.T) {

	// Test updates are committed to storage
	////////////////////////////////////////
	storage := NewMemoryStorage()
	cache := NewStorageProxyCache(storage, 10000)

	cache.SetHeight(50)
	cache.Update("secret", 999)
	cacheHasBalance(t, cache, "secret", 999)
	cacheHasUncommittedLen(t, cache, 1)
	storageHasBalance(t, storage, "secret", 0)
	
	cache.Commit()
	cacheHasUncommittedLen(t, cache, 0)
	cacheHasBalance(t, cache, "secret", 999)
	storageHasBalance(t, storage, "secret", 999)
	storageHasLen(t, storage, 1)
	
	cache.CacheClear()
	cacheHasBalance(t, cache, "secret", 999)
	storageHasBalance(t, storage, "secret", 999)

	// Reuse initialized storage for a new cache
	/////////////////////////////////////////////
	cache = NewStorageProxyCache(storage, 10000)
	cacheHasHeight(t, cache, 50)
	cacheHasBalance(t, cache, "secret", 999)
	cacheHasLen(t, cache, 1)
	cacheHasUncommittedLen(t, cache, 0)

	// Test adresses with 0 balance are deleted from storage
	////////////////////////////////////////////////////////
	storage = NewMemoryStorage()
	cache = NewStorageProxyCache(storage, 10000)

	cache.SetHeight(33)
	cache.Update("an_address", 66)
	cache.Commit()
	storageHasLen(t, storage, 1)
	storageHasBalance(t, storage, "an_address", 66)

	cache.Update("an_address", -66)
	cache.Commit()
	cacheHasBalance(t, cache, "an_address", 0)
	storageHasLen(t, storage, 0)
	storageHasBalance(t, storage, "and_address", 0)

	// Test 0 updates are discarded
	///////////////////////////////////////
	storage = NewMemoryStorage()
	cache = NewStorageProxyCache(storage, 10000)
	cache.SetHeight(100)

	// Add a new update
	cache.Update("address", 1)
	cacheHasUncommittedLen(t, cache, 1)

	// Cancel the update and check it is discarded
	cache.Update("address", -1)
	cacheHasUncommittedLen(t, cache, 0)

	cache.Commit()

	// Check nothing was added to storage
	storageHasLen(t, storage, 0)
	cacheHasLen(t, cache, 0)
	cacheHasHeight(t, cache, 100)
}

// Test max cache size after Commit
func TestStorageProxyCacheSize(t *testing.T) {
	storage := NewMemoryStorage()
	cacheSize := 1000
	storageInit(storage, cacheSize, 477777)
	cache := NewStorageProxyCache(storage, cacheSize)

	// Original cache max size before commit
	cacheHasSize(t, cache, cacheSize)

	// Add update commit and test cache
	for i := int64(0); i < int64(cacheSize+1); i++ {
		address := fmt.Sprintf("test_size_address_%v", i)
		cache.Get(address)
	}

	cache.Update("address_1", 5)
	cache.Update("address_2", 5)
	cache.Update("address_5000", 40)
	cache.Commit()
	
	cacheHasSize(t, cache, cacheSize)
}

// Test CacheLen method
func TestStorageProxyCacheLen(t *testing.T) {	
	storage := NewMemoryStorage()
	cacheSize := 1000
	storageInit(storage, cacheSize, 477777)
	cache := NewStorageProxyCache(storage, cacheSize)
	
	if cache.CacheLen() != 0 {
		t.Error("An empty cache lenght should be 0")
	}

	for i := 0; i < cacheSize; i++ {
		address := fmt.Sprintf("test_size_address_%v", i)
		cache.Get(address)
		if length := cache.CacheLen(); length != i+1 {
			t.Error("Unexpected cache size", i+1, length)
		}
	}

}

// Test negative balance detection
func TestStorageProxyNegativeBalanceDetection(t *testing.T) {	
	storage := NewMemoryStorage()
	cache := NewStorageProxyCache(storage, 100)

	cache.Update("new_address", 34)
	cache.Commit()

	cache.Update("new_address", -44)
	if err := cache.Commit(); err == nil {
		t.Error("Committing a negative should have returned and error")
	} else {
		switch err.(type) {
			case NegativeBalanceError, *NegativeBalanceError:
				return
			default:
				t.Error("Didn't return expected NegativeBalanceError")
		}
	}
}

// TODO: Test concurrent Get requests
func TestStorageProxyConcurrentGet(t *testing.T) {
	storage := NewMemoryStorage()
	cache := NewStorageProxyCache(storage, 10000)
	balance, err := cache.Get("qwerty")
	
	if err != nil {
		t.Error(balance, err)
	}

	// TODO: Retrieve concurrently the same address by 1000 routines

}

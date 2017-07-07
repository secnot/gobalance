package balance

import (
	"testing"
)


func TestStorageProxyBase(t *testing.T) {
	storage := NewMemoryStorage()
	cache := NewStorageProxyCache(storage, 10000)

	cache.Update("qwerty", 64)
}

//
func TestStorageProxyConcurrentGet(t *testing.T) {
	storage := NewMemoryStorage()
	cache := NewStorageProxyCache(storage, 10000)
	balance, ok := cache.Get("qwerty")
	t.Error(balance, ok)


	// TODO: Retrieve concurrently the same address by 1000 routines
}

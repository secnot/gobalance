package storage

import (
	"sync"
	"fmt"
	"testing"
	"reflect"
)


// Mock error storage that returns error with every method call
type MockErrorStorage struct {
	sync.Mutex
}


func NewMockErrorStorage() (storage *MockErrorStorage){
	return &MockErrorStorage{}
}

func (e *MockErrorStorage) Len() (length int, err error) {
	return 0, NewStorageError("Len()")
}

func (e *MockErrorStorage) GetHeight() (height int64, err error) {
	// Doesn't return error so proxy cache can be created
	return 0, nil
}

func (e *MockErrorStorage) SetHeight(height int64) (err error) {
	return NewStorageError("SetHeight()")
}

func (e *MockErrorStorage) Get(address string) (value int64, err error) {
	return 55, NewStorageError("Get()")
}

func (e *MockErrorStorage) Set(address string, value int64) (err error) {
	return NewStorageError("Set()")
}

func (e *MockErrorStorage) Update(address string, value int64) (err error) {
	return NewStorageError("Update()")
}

func (e *MockErrorStorage) Delete(address string) (err error) {
	return NewStorageError("Delete()")
}

func (e *MockErrorStorage) Contains(address string) (contains bool, err error) {
	return false, NewStorageError("Contains()")
}

func (e *MockErrorStorage) BulkGet(addresses []string) (balance []int64, err error) {
	return nil, NewStorageError("MockBulkGetError")
}

func (e *MockErrorStorage) BulkUpdate(update []AddressBalancePair, 
			   height int64) (err error) {
	return NewStorageError("MockBulkUpdateError")
}


// Test storage returning error while calling Get
func TestStorageProxyCacheStorageErrorGet(t *testing.T) {

	storage := NewMockErrorStorage()
	cache, err := NewStorageProxyCache(storage, 1000)
	if err != nil {
		t.Error("Unable to create StorageProxyCache")
	}

	value, err := cache.Get("random_address")
	if err == nil {
		t.Error("Get(): Expected an error")
	}
	if value != 0 {
		t.Error("Get() returned value should be 0 when there is an error")
	}

	switch e := err.(type) {
	case StorageError, *StorageError:
		return
	default:
		t.Error(fmt.Sprintf("Get(): %v %v", reflect.TypeOf(e), e))
	}
}

// Test storage returning error while calling Commit
func TestStorageProxyCacheStorageErrorCommit(t *testing.T) {	
	
	storage := NewMockErrorStorage()
	cache, err := NewStorageProxyCache(storage, 1000)
	if err != nil {
		t.Error("Unable to create StorageProxyCache")
	}

	cache.Update("random_address", 44)
	err = cache.Commit()
	switch e := err.(type) {
	case StorageError, *StorageError:
		return
	default:
		t.Error(fmt.Sprintf("Get(): %v %v", reflect.TypeOf(e), e))
	}

}


func TestNewStorageError(t *testing.T) {
	err := NewStorageError("message")
	if err.Error() != "message" {
		t.Error("NewStorageError() Didn't return the correct message")
	}
}


func TestNegativeBalanceError(t *testing.T) {
	err := NewNegativeBalanceError("message")
	if err.Error() != "message" {
		t.Error("NewNegativeStorageError() Didn't return the correct message")
	}
}

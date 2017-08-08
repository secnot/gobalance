package storage

import (
	"sync"
	"fmt"
)

// In Memory Balance Storage
type MemoryStorage struct {
	sync.RWMutex

	// Height for the last block stored, -1 when there's none
	height int64

	// Address:Balance storage
	store map[string]int64
}

// New Memory Storage
func NewMemoryStorage() (*MemoryStorage) {
	storage := MemoryStorage{
		height: -1,
		store: make(map [string]int64),
	}
	return &storage
}

// Get stored height
func (s *MemoryStorage) GetHeight() (height int64, err error){
	s.RLock()
	height, err = s.height, nil
	s.RUnlock()
	return
}

// Set new height
func (s *MemoryStorage) SetHeight(height int64) (err error){
	s.Lock()
	s.height = height
	s.Unlock()
	return nil
}


// Return number of stored balances
func (s *MemoryStorage) Len() (length int, err error){
	s.RLock()
	length = len(s.store)
	s.RUnlock()
	return length, nil
}

// Get address balance, return 0 if not present
func (s *MemoryStorage) Get(address string) (value int64, err error) {
	s.RLock()
	value, _ = s.store[address]
	s.RUnlock()
	return value, nil
}


// Set address balance, if 0 the value is deleted
func (s *MemoryStorage) Set(address string, balance int64) (err error){
	s.Lock()
	if balance > 0 {
		s.store[address] = balance
	} else if balance == 0  {
		delete(s.store, address)
	} else {
		errMsg := fmt.Sprintf("%v: %v", address, balance)
		err = NewNegativeBalanceError(errMsg)
	}
	s.Unlock()
	return err
}

// Update or create an address balance by adding or substracting a 
// value. If the resulting balance is 0 the record is deleted.
func (s *MemoryStorage) Update(address string, update int64) (err error) {
	s.Lock()
	balance := s.store[address]
	if balance + update > 0 {
		s.store[address] = balance + update
	} else if balance + update == 0 {
		delete(s.store, address)
	} else {
		errMsg := fmt.Sprintf("%v: %v", address, balance + update)
		err = NewNegativeBalanceError(errMsg)
	}
	s.Unlock()
	return err
}

// Contains returns true if the address is stored false otherwise 
func (s *MemoryStorage) Contains(address string) (cont bool, err error) {
	s.RLock()
	_, cont = s.store[address]
	s.RUnlock()
	return
}

// Delete removes an address from balance if it doen't exist nothing happens
func (s *MemoryStorage) Delete(address string) (err error){
	s.Lock()
	delete(s.store, address)
	s.Unlock()
	return nil
}

// BulkGet returns the balance for the address in the slice
func (s *MemoryStorage) BulkGet(addresses []string) (balance []int64, err error){

	balance = make([]int64, len(addresses))
	s.RLock()
	for n, addr := range addresses {
		if bal, ok := s.store[addr]; ok {
			balance[n] = bal
		}
	}
	s.RUnlock()
	return
}

// BulkUpdate mass updates storage balance and height in an atomic operation
func (s *MemoryStorage) BulkUpdate(update []AddressBalancePair, 
				height int64) (err error){
	s.Lock()
	// TODO: Check none of the updates result in a negative balance before 
	// updating storage so there's no need to backtrack.
	// Update existing balance
	for _, up := range update {
		currentBalance := s.store[up.Address]
		if currentBalance + up.Balance > 0 {
			s.store[up.Address] = currentBalance + up.Balance
		} else if currentBalance + up.Balance == 0 {
			delete(s.store, up.Address)
		} else {
			errMsg := fmt.Sprintf("%v: %v", up.Address, currentBalance + up.Balance)
			err = NewNegativeBalanceError(errMsg)
			break
		}
	}

	// 
	s.height = height
	s.Unlock()
	return err
}


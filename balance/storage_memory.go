package balance

import "sync"

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
	if balance == 0 {
		delete(s.store, address)
	} else {
		s.store[address] = balance
	}
	s.Unlock()
	return nil
}

// Contains returns true if the address is stored false otherwise 
func (s *MemoryStorage) Contains(address string) (cont bool, err error) {
	if _, ok := s.store[address]; !ok {
		return false, nil
	}
	return true, nil
}

// Remove address from balance
func (s *MemoryStorage) Delete(address string) (err error){
	s.Lock()
	delete(s.store, address)
	s.Unlock()
	return nil
}

// Return values for a list of addresses
func (s *MemoryStorage) BulkGet(addresses []string) (balance []int64, err error){

	values := make([]int64, len(addresses))
	s.RLock()
	for n, addr := range addresses {
		if value, ok := s.store[addr]; ok {
			values[n] = value
		}
	}
	s.RUnlock()

	return values, nil
}

// Mass update storage in an atomic operation 
// (safely update SQL DB within a single transaction)
func (s *MemoryStorage) BulkUpdate(insert []AddressBalancePair, 
							 update []AddressBalancePair, 
							 remove []string, height int64) (err error){
	s.Lock()

	// Insert new pairs
	for _, pair := range insert {	
		if pair.Balance != 0 {
			s.store[pair.Address] = pair.Balance
		}
	}

	// Update existing balance
	for _, pair := range update {
		if pair.Balance != 0 {
			s.store[pair.Address] = pair.Balance
		} else {
			delete(s.store, pair.Address)
		}
	}

	// Delete balance
	for _, address := range remove {
		delete(s.store, address)
	}

	// 
	s.height = height
	s.Unlock()
	return nil
}


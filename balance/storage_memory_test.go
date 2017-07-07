package balance

import(
	"testing"
	"fmt"
	"time"
)


// Test item in storage
func storageContains(t *testing.T, storage Storage, address string, balance int64) {
	value, err := storage.Get(address)
	if err != nil {
		t.Error(fmt.Sprintf("Get(%v) returned error", address))
		return
	}
	if value != balance {
		t.Error(fmt.Printf("Get(%v) returned %v (expected %v)", address, value, balance))
	}
}

// Test item is not stored
func storageNotContains(t *testing.T, storage Storage, address string) {
	value, err := storage.Get(address)
	if err != nil {
		t.Error(fmt.Sprintf("Get(%v) returned error", address))
		return
	}
	if value != 0 {
		t.Error(address, "shouldn't been in storage but it was")
	}

	if cont, _ := storage.Contains(address); cont {
		t.Error(address, "shouldn't been in storage but it was")
	}
}

// Storage is empty
func storageLengthIs(t *testing.T, storage Storage, size int) {

	if retSize, err := storage.Len(); err != nil {
		t.Error("Len() returned an error")
	} else if retSize != size {
		t.Error("Unexpected storage length", retSize)
	}
}

// Some basic storage iterface tests 
func testStorageInterfaceBase(t *testing.T, storage Storage) {

	// Check it's empty
	storageLengthIs(t, storage, 0)

	// Check heigh is not initialized
	if height, err := storage.GetHeight(); height != -1 || err != nil {
		t.Error("Height shouldn't be initialized")
	}

	// Set new height
	if err := storage.SetHeight(33); err != nil {
		t.Error("There was an error while setting a new height")
	}

	if height, err := storage.GetHeight(); height != 33 || err != nil {
		t.Error("Height wasn't updated should be 33 but it's", height)
	}

	// Unknown address balance is always 0
	storageNotContains(t, storage, "unknown address")

	// Set value and check
	if err := storage.Set("address", 55); err != nil {
		t.Error("There was an error while setting address balance")
	}
	storageContains(t, storage, "address", 55)
	storageLengthIs(t, storage, 1)

	// Update value
	if err := storage.Set("address", 100); err != nil {
		t.Error("There was an error while setting address balance")
	}
	storageContains(t, storage, "address", 100)

	// Test deleting existing address balance
	if err := storage.Delete("address"); err != nil {
		t.Error("There was an error while deleting and address")
	}
	storageNotContains(t, storage, "address")

	// Test deleting unknown address isn't an error
	storage.Delete("an unknown address")
	if bal, err := storage.Get("an unknown address"); bal != 0 || err != nil {
		t.Error("Something happened while deleting")
	}

	// Test setting an address balance to 0 removes it from storage
	initialLength, _ := storage.Len()
	storage.Set("random_address", 12)
	storageLengthIs(t, storage, initialLength+1)
	storage.Set("random_address", 0)
	storageLengthIs(t, storage, initialLength)

	// Test contains method
	storage.Set("one_address", 12)
	if cont, _ := storage.Contains("one_address"); !cont {
		t.Error("storage should contain \"one_address\"")	
	}
	if cont, _ := storage.Contains("random_address"); cont {
		t.Error("storage shouldn't contain \"random_address\"")	
	}
}



// Test Bulk get address balance 
func testStorageInterfaceBulkGet(t *testing.T, storage Storage) {
	
	for balance := int64(1); balance < 1000; balance++ {
		addr := fmt.Sprintf("address%v", balance)
		storage.Set(addr, balance)
	}

	// Get existing
	addr := []string{"address1", "address2", "address100"}
	balance, err := storage.BulkGet(addr)
	if err != nil {
		t.Error("There was an error while callint BulkGet")
	}
	if balance[0] != 1 || balance[1] != 2 || balance[2] != 100 {
		t.Error("Wrong bulk balance returned")
	}

	// Get Not existing
	addr = []string{"address0", "address1500", "address1600"}
	balance, err = storage.BulkGet(addr)
	if err != nil {
		t.Error("There was an error while requesting non-existant address balance")
	}
	if balance[0] != 0 || balance[1] != 0 || balance[2] != 0 {
		t.Error("Wrong bulk balance returned")
	}

	// Get mixed 
	addr = []string{"address2000", "address900", "address10", "other"}
	bal, err := storage.BulkGet(addr)
	if err != nil {
		t.Error("There was an error during BulkGet call")
	}
	if bal[0] != 0 || bal[1] != 900 || bal[2] != 10 || bal[3] != 0 {
		t.Error("Wrong bulk balance returned")
	}

	// Get empty slice
	addr = []string{}
	bal, err = storage.BulkGet(addr)
	if err != nil {
		t.Error("There was an error with the empty BulkGet call")
	}
	if len(bal) != 0 {
		t.Error("An empty request shouldn't return any balance")
	}
}


// Test mass updating
func testStorageInterfaceBulkUpdate(t *testing.T, storage Storage) {
	
	for balance := int64(1); balance < 1000; balance++ {
		addr := fmt.Sprintf("address%v", balance)
		storage.Set(addr, balance)
	}

	// update+insert+remove
	update := []AddressBalancePair{
		{ Address: "address1", Balance: 100 },
		{ Address: "address2", Balance: 200 },
	}
	insert := []AddressBalancePair{
		{ Address: "new_address1", Balance: 1 },
		{ Address: "new_address2", Balance: 2 },
	}

	remove := []string{"address10", "address20"}
	height := int64(100)

	if err := storage.BulkUpdate(insert, update, remove, height); err != nil {
		t.Error("There was an error during BulkUpdate call")
	}

	// Check updated
	storageContains(t, storage, "address1", 100)
	storageContains(t, storage, "address2", 200)

	// Check inserts
	storageContains(t, storage, "new_address1", 1)
	storageContains(t, storage, "new_address2", 2)

	// Check removed
	storageNotContains(t, storage, "address10")
	storageNotContains(t, storage, "address20")
	
	// Check height
	if h, err := storage.GetHeight(); h != height || err != nil{
		t.Error("Height wasn't updated")
	}

	// Test empty update
	for balance := int64(1); balance < 1000; balance++ {
		addr := fmt.Sprintf("address%v", balance)
		storage.Set(addr, balance)
	}

	prevLength, _ := storage.Len()
	storage.BulkUpdate(nil, nil, nil, height)
	for balance := int64(1); balance < 1000; balance++ {
		addr := fmt.Sprintf("address%v", balance)	
		storageContains(t, storage, addr, balance)
	}
	storageLengthIs(t, storage, prevLength)
}


// Test concurrent Set/Get
func testStorageInterfaceConcurrency(t *testing.T, storage Storage) {

	updateFunc := func (storage Storage, start int, end int) {
		
		for i:=start; i<end; i++ {
			time.Sleep(2*time.Millisecond)
			addr := fmt.Sprintf("address%v", i)
			storage.Set(addr, int64(i))
		
			// Also check previous balance
			if i > start {
				addr := fmt.Sprintf("address%v", i-1)
				storage.Get(addr)
			}
		}
	}

	go updateFunc(storage, 1, 1000)
	go updateFunc(storage, 1, 1000)
	go updateFunc(storage, 1, 1000)
	go updateFunc(storage, 1, 1000)
	go updateFunc(storage, 2000, 3000)
	go updateFunc(storage, 2000, 3000)
	go updateFunc(storage, 2000, 3000)
	go updateFunc(storage, 2000, 3000)

	// Wait until all updates are finished
	time.Sleep(10*time.Second)

	// Check updated values	
	for balance := int64(1); balance < 1000; balance++ {
		addr := fmt.Sprintf("address%v", balance)
		storageContains(t, storage, addr, balance)		
	}

	for balance := int64(2000); balance < 3000; balance++ {
		addr := fmt.Sprintf("address%v", balance)
		storageContains(t, storage, addr, balance)		
	}
}



//	MemoryStorage tests
//////////////////////////

// Some very basic memory storage tests 
func TestMemoryStorageBase(t *testing.T) {
	storage := NewMemoryStorage()
	testStorageInterfaceBase(t, storage)
}

// Test memory storage BulkGet
func TestMemoryStorageBulkGet(t *testing.T) {
	storage := NewMemoryStorage()
	testStorageInterfaceBulkGet(t, storage)
}

// Test memory storage BulkUpdate
func TestMemoryStorageBulkUpdate(t *testing.T) {
	storage := NewMemoryStorage()
	testStorageInterfaceBulkUpdate(t, storage)
}

// Test memory storage concurrency
func TestMemoryStorageConcurrency(t *testing.T) {
	storage := NewMemoryStorage()
	testStorageInterfaceConcurrency(t, storage)
}




// TODO
//  SQLiteStorage tests
/////////////////////////





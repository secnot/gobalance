package storage

import(
	"testing"
	"fmt"
	"github.com/secnot/gobalance/primitives"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func max(a, b int) int {

	if a >= b {
		return a
	} else {
		return b
	}
}


// Test item in storage
func storageContains(t *testing.T, storage Storage, out primitives.TxOut) {

	requested := TxOutId{TxHash: *out.TxHash, Nout: out.Nout}
	data, err := storage.Get(requested)
	if err != nil {
		t.Errorf("Get(%v): %v", out, err)
		return
	}

	expected := TxOutData{Addr: out.Addr, Value: out.Value}
	if data != expected {
		t.Errorf("Get(%v) returned %v (expecting %v)", requested, data, expected)
	}
}

// Test item is not stored
func storageNotContains(t *testing.T, storage Storage, out TxOutId) {
	data, err := storage.Get(out)
	if err != nil {
		t.Errorf("Get(%v): %v", out, err)
		return
	}

	defaultTxOut := TxOutData{Addr: "", Value: 0}
	if data != defaultTxOut {
		t.Error(out, "shouldn't be in storage")
	}

	if cont, _ := storage.Contains(out); cont {
		t.Error(out, "shouldn't be in storage")
	}
}

// Check stored height
func storageHeightIs(t *testing.T, storage Storage, height int64) {
	stoHeight, err := storage.GetHeight()
	if err != nil {
		t.Error("GetHeight(): ", err)
		return
	}

	if stoHeight != height {
		t.Error("GetHeight(): returned %v expecting %v", stoHeight, height) 
	}
}

// Check Storage length
func storageLengthIs(t *testing.T, storage Storage, size int) {

	if retSize, err := storage.Len(); err != nil {
		t.Error("Len():", err)
	} else if retSize != size {
		t.Errorf("Storage length is %v expecting %v", retSize, size)
	}
}

// mockHash
func mockHash(id uint) chainhash.Hash {
	hashStr := fmt.Sprintf("txhash_%v", id)
	var hash chainhash.Hash
	copy(hash[:], hashStr)
	return hash
}

// mockAddress
func mockAddress(id uint) string {
	return fmt.Sprintf("address_%v", id)
}

// mockTxOuts returns an array of initialized records (start included, end excluded)
func mockTxOuts(start uint, end uint, nouts uint, value_increment int) []primitives.TxOut {	
	if nouts < 1 {
		nouts = 1
	}
	if end < start {
		panic("TxOutRecords start > end")
	}

	records := make([]primitives.TxOut, 0, (end-start)*nouts)
	for i := start; i < end; i++ {
		hash := mockHash(i)
		addr := mockAddress(i)
		
		for out := uint(0); out < nouts; out++ {
			record := primitives.TxOut{
				TxHash: &hash, 
				Nout: uint32(out), 
				Addr: addr, 
				Value: int64(i)+int64(value_increment)}
			records = append(records, record)
		}
	}

	return records
}

// TxOutDataToId creates a TxOutId slice from a TxOutRecord slice
func TxOutToId(outs []primitives.TxOut) []TxOutId {

	ids := make([]TxOutId, len(outs))
	for n, out := range outs {
		ids[n] = TxOutId{TxHash: *out.TxHash, Nout: out.Nout}
	}

	return ids
}

// TxOutToData creates a TxOutData slice from a TxOutRecord slice
func TxOutToData(outs []primitives.TxOut) []TxOutData {

	data := make([]TxOutData, len(outs))
	for n, out := range outs {
		data[n] = TxOutData{Addr: out.Addr, Value: out.Value}
	}

	return data
}

// Initialize storage with mock txout (start included, end not)
func initStorage(t *testing.T, storage Storage, records []primitives.TxOut) {
	
	for _, txout := range records {
		err := storage.Set(txout)
		if err != nil {
			t.Errorf("initStorage(): %v", err)
			return
		}
	}
}

// Test GetHeight method
func TestSQLiteGetHeight(t *testing.T) {


	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	// Test default height for uninitialized height
	height, err := storage.GetHeight()
	if err != nil {
		t.Error("GetHeight(): ", err)
		return
	}
	if height != -1 {
		t.Error("GetHeight(): Un initialized height should be -1 not ", height)
		return
	}

	err = storage.SetHeight(10)
	if err != nil {
		t.Error("SetHeight(): ", err)
		return
	}
	
	height, err = storage.GetHeight()
	if err != nil || height != 10 {
		t.Errorf("GetHeight(): Expecting %v returned %v", err, height)
		return
	}
	
}

// Test SetHeight method
func TestSQLiteSetHeight(t *testing.T) {

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	err = storage.SetHeight(10)
	if err != nil {
		t.Error("SetHeight(): ", err)
		return
	}
	
	height, err := storage.GetHeight()
	if err != nil || height != 10 {
		t.Errorf("GetHeight(): Expecting %v returned %v", err, height)
		return
	}

	// Test error on setting negative height
	err = storage.SetHeight(-10)
	if err != ErrNegativeHeight {
		t.Error("SetHeight(): Expecting ErrNegativeHeight not", err)
		return
	}

	// Check failed SetHeight didn't change height	
	height, err = storage.GetHeight()
	if err != nil || height != 10 {
		t.Errorf("GetHeight(): Expecting %v returned %v", err, height)
		return
	}
}

// Test Get method
func TestSQLiteGet(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	mockRecords := mockTxOuts(0, 10, 1, 0)
	mockIds     := TxOutToId(mockRecords)
	mockData    := TxOutToData(mockRecords)
	
	storage.Set(mockRecords[1])
	data, err := storage.Get(mockIds[1])
	if err != nil {
		t.Error("Get(): ", err)
		return
	}
	if data != mockData[1] {
		t.Errorf("Get(): Expecting %v returned %v", mockData[1], data)
		return
	}

	// Check get returns default TxOutCache for missing utxo
	fakeUtxoId := TxOutId{TxHash: mockHash(7777777), Nout: 7777}
	defaultData := TxOutData{Addr: "", Value: int64(0)}
	data, err = storage.Get(fakeUtxoId)
	if err != nil {
		t.Error("Get(): ", err)
		return
	}
	if data != defaultData {
		t.Errorf("Get(): Expecting %v returned %v", defaultData, data)
		return
	}
}

// Test Set method
func TestSQLiteSet(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	mockOuts := mockTxOuts(0, 10, 1, 0)

	err = storage.Set(mockOuts[1])
	storageContains(t, storage, mockOuts[1])
	storageLengthIs(t, storage, 1)
	
	err = storage.Set(mockOuts[2])
	storageContains(t, storage, mockOuts[2])
	storageLengthIs(t, storage, 2)

	// Test inserting a negative value utxo raise an error
	hash := mockHash(45634523)
	err = storage.Set(primitives.TxOut{TxHash: &hash, Nout: 2, Addr: "Address", Value: -1})
	if err == nil || err != ErrNegativeUtxo {
		t.Error("Set(): Expecting an ErrNegativeUtxo error not", err)
		return
	}

	// Test inserting an unexpendable utxo raise an error
	err = storage.Set(primitives.TxOut{TxHash: &hash, Nout: 2, Addr: "Address", Value: 0})
	if err == nil || err != ErrUnexpendableUtxo {
		t.Error("Set(): Expecting an ErrNegativeUtxo error not", err)
		return
	}
	
	err = storage.Set(primitives.TxOut{TxHash: &hash, Nout: 2, Addr: "", Value: 10})
	if err == nil || err != ErrUnexpendableUtxo {
		t.Error("Set(): Expecting an ErrNegativeUtxo error not", err)
		return
	}

	// Test Overwritting an existing utxo raises an error
	hash1 := mockHash(456388345)
	insert := primitives.TxOut{TxHash: &hash1, Nout: 1, Addr: "address1", Value: 10}
	update := primitives.TxOut{TxHash: &hash1, Nout: 1, Addr: "address1", Value: 20}
	err = storage.Set(insert)
	err = storage.Set(update)
	if err == nil {
		t.Error("UNIQUE constraint failed: utxo.tx, utxo.nout")
	}
	storageContains(t, storage, insert)
}

// Test BulkGet method
func TestSQLiteBulkGet(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	
	storedOuts := mockTxOuts(10000, 100000, 2, 0)

	queryOuts := mockTxOuts(10000, 60000, 2, 0)
	queryData := TxOutToData(queryOuts)
	queryId   := TxOutToId(queryOuts)
	
	initStorage(t, storage, storedOuts)

	for test := 0; test <2; test++ {
		data, err := storage.BulkGet(queryId)
		if err != nil {
			t.Error(err)
			return
		}

		for n, _ := range queryData {
			if data[n] != queryData[n] {
				t.Errorf("Expected %v, returned %v", queryData[n], data[n])
				return
			}
		}
	}

	// Test it returns default TxOut for missing utxo
	defaultTxOut := TxOutData{Addr: "", Value: int64(0)}
	missingOuts := mockTxOuts(1000000, 1000010, 2, 0)
	returnedOuts, err := storage.BulkGet(TxOutToId(missingOuts))
	if err != nil {
		t.Error("BulkGet(): ", err)
		return
	}
	for _, utxo := range returnedOuts {
		if utxo != defaultTxOut {
			t.Error("BulkGet(): Expected default TxOutData, not", utxo)
		}
	}
}

// Test BulkUpdate method
func TestSQLiteBulkUpdate(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("Error initializing DB")
		return
	}
	
	storedOuts := mockTxOuts(10000, 100000, 2, 0)
	initStorage(t, storage, storedOuts)
	storageHeightIs(t, storage, int64(-1))
	storageLengthIs(t, storage, 180000)

	//
	removeOuts := mockTxOuts(10000, 20000, 2, 0)
	insertOuts := mockTxOuts(200000, 207500, 2, 0)

	removeIds := TxOutToId(removeOuts)

	//
	storage.BulkUpdate(insertOuts, removeIds, int64(10000))
	storageHeightIs(t, storage, int64(10000))
	storageLengthIs(t, storage, 175000)
	
	// Compare inserted records
	for _, ins := range insertOuts {
		storageContains(t, storage, ins)
	}

	// Check deleted records
	for _, rem := range removeIds {
		storageNotContains(t, storage, rem)
	}
}

// Test BulkUpdate return errors for invalid operations
func TestSQLiteBulkUpdateErrors(t *testing.T) {	
	
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("Error initializing DB")
		return
	}
	
	storedOuts := mockTxOuts(1, 100000, 2, 0)
	initStorage(t, storage, storedOuts)
	storageHeightIs(t, storage, int64(-1))
	storageLengthIs(t, storage, 199998)

	hash1 := mockHash(11111111)
	hash2 := mockHash(22222222)
	insertNegativeOuts     := mockTxOuts(200000, 200001, 1, -1000000)
	insertZeroOuts         := [...]primitives.TxOut{{TxHash: &hash2, Nout: 1, Addr: "addr2", Value: 0}}
	insertEmptyAddressOuts := [...]primitives.TxOut{{TxHash: &hash1, Nout: 1, Addr: "", Value: 10}}

	//  Test negative detection
	err = storage.BulkUpdate(insertNegativeOuts, nil, 10)
	if err == nil || err != ErrNegativeUtxo {
		t.Error("BulkUpdate(): Expecting an ErrNegativeUtxo error not", err)
		return
	}

	// Test unexpendable detection
	err = storage.BulkUpdate(insertEmptyAddressOuts[:], nil, 20)
	if err == nil || err != ErrUnexpendableUtxo {
		t.Error("BulkUpdate(): Expecting an ErrUnexpendableUtxo error not", err)
		return
	}
	
	err = storage.BulkUpdate(insertZeroOuts[:], nil, 20)
	if err == nil || err != ErrUnexpendableUtxo {
		t.Error("BulkUpdate(): Expecting an ErrUnexpendableUtxo error not", err)
		return
	}

	// Test updating stored utxo
	initial := [...]primitives.TxOut{{TxHash: &hash2, Nout: 1, Addr: "addr2", Value: 10}}
	updated := [...]primitives.TxOut{{TxHash: &hash2, Nout: 1, Addr: "addr2", Value: 20}}
	err = storage.BulkUpdate(initial[:], nil, 100)
	if err != nil {
		t.Error("BulkUpdate(): Unexpected error")
		return
	}
	err = storage.BulkUpdate(updated[:], nil, 2000)
	if err == nil {
		t.Error("BulkUpdate(): UNIQUE constraint failed: utxo.tx, utxo.nout")
		return
	}

	// Deleting a non existent record is not an error
	hash3 := mockHash(33333333)
	remove := [...]primitives.TxOut{{TxHash: &hash3, Nout: 100, Addr: "addr2", Value: 20}}
	err = storage.BulkUpdate(nil, TxOutToId(remove[:]), 8000)
	if err != nil {
		t.Error("BulkUpdate(): Unexpected error while deleting non existent utxo", err)
		return
	}
}


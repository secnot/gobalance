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
func storageLastBlockIs(t *testing.T, storage Storage, height int64, hash chainhash.Hash) {
	stoHeight, stoHash, err := storage.GetLastBlock()
	if err != nil {
		t.Error("GetLastBlock(): ", err)
		return
	}

	if stoHeight != height {
		t.Error("GetLastBlock(): returned %v expecting %v", stoHeight, height) 
	}

	if stoHash != hash {
		t.Error("GetLastBlock(): returned %v expecting %v", stoHash, hash)
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

// Test GetLastBlock method
func TestSQLiteGetLastBlock(t *testing.T) {


	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	// Test default height and hash for uninitialized DB
	height, hash, err := storage.GetLastBlock()
	if err != nil {
		t.Error("GetLastBlock(): ", err)
		return
	}
	if height != -1 {
		t.Error("GetLastBlock(): Uninitialized height should be -1 not ", height)
		return
	}
	if hash != primitives.ZeroHash {
		t.Errorf("GetLastBlock(): Uninitialize hash should be %v not %v", 
				primitives.ZeroHash, hash)
		return
	}

	// Set and check last block
	err = storage.SetLastBlock(10, primitives.MainNetGenesisHash)
	if err != nil {
		t.Error("SetLastBlock(): ", err)
		return
	}
	
	height, hash, err = storage.GetLastBlock()	
	if err != nil {
		t.Error("GetLastBlock(): ", err)
		return
	}
	if height != 10 {
		t.Error("GetLastBlock(): Uninitialized height should be 10 not ", height)
		return
	}
	if hash != primitives.MainNetGenesisHash {
		t.Errorf("GetLastBlock(): Uninitialize hash should be %v not %v", 
				primitives.MainNetGenesisHash, hash)
	}


}

// Test SetLastBlock method
func TestSQLiteSetLastBlock(t *testing.T) {

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	// Test returns error for negative heights
	err = storage.SetLastBlock(-10, primitives.MainNetGenesisHash)
	if err != ErrNegativeHeight {
		t.Error("SetLastBlock(): Expecting ErrNegativeHeight not", err)
		return
	}

	// Check failed SetLastBlock didn't change height or hash
	height, hash, err := storage.GetLastBlock()
	if err != nil || height != -1 || hash != primitives.ZeroHash {
		t.Error("GetLastBlock(): Unexpected modifications")
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

// Test GetBalance method
func TestSQLiteGetBalance(t *testing.T) {	
	
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	
	storedOuts := mockTxOuts(1000, 1010, 1, 0)
	storedOuts[0].Addr  = "some_address"
	storedOuts[0].Value = 1
	storedOuts[1].Addr  = "some_address"
	storedOuts[1].Value = 2
	storedOuts[2].Addr  = "other_address"
	storedOuts[2].Value = 5
	initStorage(t, storage, storedOuts)

	// Test address not in storage
	balance, err := storage.GetBalance("unknown_address")
	if err != nil {
		t.Error("GetBalance(): ", err)
	}
	if balance != 0 {
		t.Error("GetBalance(): Expecting 0 returned ", balance)
	}

	// Test address with a single utxout
	balance, err = storage.GetBalance("other_address")
	if err != nil {
		t.Error("GetBalance(): ", err)
	}
	if balance != 5 {
		t.Error("GetBalance(): Expecting 3 returned ", balance)
	}

	// Test address with more than one utxout
	balance, err = storage.GetBalance("some_address")
	if err != nil {
		t.Error("GetBalance(): ", err)
	}
	if balance != 3 {
		t.Error("GetBalance(): Expecting 3 returned ", balance)
	}
}

// Test GetByAddress method
func TestSQLiteGetByAddress(t *testing.T) {	
	
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	
	
	storedOuts := mockTxOuts(1000, 1010, 1, 0)
	storedOuts[0].Addr = "some_address"
	storedOuts[1].Addr = "some_address"
	storedOuts[2].Addr = "other_address"
	initStorage(t, storage, storedOuts)

	// Get unknown address
	outs, err := storage.GetByAddress("missing_address")
	if err!= nil {
		t.Error("GetByAddress(): ", err)
	}
	if len(outs) != 0 {
		t.Errorf("GetByAddress(0): Returned %v utxo", len(outs))
	}

	// Get address with a single utxo
	outs, err = storage.GetByAddress("other_address")
	if err!= nil {
		t.Error("GetByAddress(): ", err)
		return
	}
	if len(outs) != 1 {
		t.Errorf("GetByAddress(1): Returned %v utxo", len(outs))
		return
	}
	if *outs[0].TxHash != *storedOuts[2].TxHash {
		t.Error("GetByAddress(1): Unexpected hash", *outs[0].TxHash)
		return
	}
	if outs[0].Value != storedOuts[2].Value {
		t.Error("GetByAddress(1): Unexpected value", outs[0].Value)
		return
	}
	if outs[0].Addr != storedOuts[2].Addr {
		t.Error("GetByAddress(1): Unexpected address", outs[0].Addr)
		return
	}

	// Get address with two utxo
	outs, err = storage.GetByAddress("some_address")
	if err!= nil {
		t.Error("GetByAddress(): ", err)
	}
	if len(outs) != 2 {
		t.Errorf("GetByAddress(2): Returned %v utxo", len(outs))
	}

	var out0, out1 primitives.TxOut
	if *outs[0].TxHash == *storedOuts[0].TxHash {
		out0 = outs[0]
		out1 = outs[1]
	} else {
		out0 = outs[1]
		out1 = outs[0]
	}
	if *out0.TxHash != *storedOuts[0].TxHash || *out1.TxHash != *storedOuts[1].TxHash {
		t.Error("GetByAddress(2): Unexpected Txhash")
		return
	}
	if out0.Value != storedOuts[0].Value || out1.Value != storedOuts[1].Value {
		t.Error("GetByAddress(2): Unexpected value")
		return
	}
	if out0.Addr != storedOuts[0].Addr || out1.Addr != storedOuts[1].Addr {
		t.Error("GetByAddress(2): Unexpected address")
		return
	}
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
	storageLastBlockIs(t, storage, int64(-1), primitives.ZeroHash)
	storageLengthIs(t, storage, 180000)

	//
	removeOuts := mockTxOuts(10000, 20000, 2, 0)
	insertOuts := mockTxOuts(200000, 207500, 2, 0)

	removeIds := TxOutToId(removeOuts)

	//
	storage.BulkUpdate(insertOuts, removeIds, int64(10000), primitives.MainNetGenesisHash)
	storageLastBlockIs(t, storage, int64(10000), primitives.MainNetGenesisHash)
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
	storageLastBlockIs(t, storage, int64(-1), primitives.ZeroHash)
	storageLengthIs(t, storage, 199998)

	hash1 := mockHash(11111111)
	hash2 := mockHash(22222222)
	insertNegativeOuts     := mockTxOuts(200000, 200001, 1, -1000000)
	insertZeroOuts         := [...]primitives.TxOut{{TxHash: &hash2, Nout: 1, Addr: "addr2", Value: 0}}
	insertEmptyAddressOuts := [...]primitives.TxOut{{TxHash: &hash1, Nout: 1, Addr: "", Value: 10}}

	//  Test negative detection
	err = storage.BulkUpdate(insertNegativeOuts, nil, 10, primitives.MainNetGenesisHash)
	if err == nil || err != ErrNegativeUtxo {
		t.Error("BulkUpdate(): Expecting an ErrNegativeUtxo error not", err)
		return
	}

	// Test unexpendable detection
	err = storage.BulkUpdate(insertEmptyAddressOuts[:], nil, 20, primitives.MainNetGenesisHash)
	if err == nil || err != ErrUnexpendableUtxo {
		t.Error("BulkUpdate(): Expecting an ErrUnexpendableUtxo error not", err)
		return
	}
	
	err = storage.BulkUpdate(insertZeroOuts[:], nil, 20, primitives.MainNetGenesisHash)
	if err == nil || err != ErrUnexpendableUtxo {
		t.Error("BulkUpdate(): Expecting an ErrUnexpendableUtxo error not", err)
		return
	}

	// Test updating stored utxo
	initial := [...]primitives.TxOut{{TxHash: &hash2, Nout: 1, Addr: "addr2", Value: 10}}
	updated := [...]primitives.TxOut{{TxHash: &hash2, Nout: 1, Addr: "addr2", Value: 20}}
	err = storage.BulkUpdate(initial[:], nil, 100, primitives.MainNetGenesisHash)
	if err != nil {
		t.Error("BulkUpdate(): Unexpected error")
		return
	}
	err = storage.BulkUpdate(updated[:], nil, 2000, primitives.ZeroHash)
	if err == nil {
		t.Error("BulkUpdate(): UNIQUE constraint failed: utxo.tx, utxo.nout")
		return
	}

	// Deleting a non existent record is not an error
	hash3 := mockHash(33333333)
	remove := [...]primitives.TxOut{{TxHash: &hash3, Nout: 100, Addr: "addr2", Value: 20}}
	err = storage.BulkUpdate(nil, TxOutToId(remove[:]), 8000, primitives.ZeroHash)
	if err != nil {
		t.Error("BulkUpdate(): Unexpected error while deleting non existent utxo", err)
		return
	}
}


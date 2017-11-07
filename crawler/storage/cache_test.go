package storage

import (
	"testing"
	"github.com/secnot/gobalance/primitives"
)

// Test item in storage
func cacheContains(t *testing.T, cache *StorageCache, out primitives.TxOut) {

	requested := TxOutId{TxHash: *out.TxHash, Nout: out.Nout}
	data, err := cache.GetTxOut(requested)
	if err != nil {
		t.Errorf("GetTxOut(%v): %v", out, err)
		return
	}

	expected := TxOutData{Addr: out.Addr, Value: out.Value}
	if data != expected {
		t.Errorf("GetTxOut(%v) returned %v (expecting %v)", requested, data, expected)
	}
}

// Test item is not stored
func cacheNotContains(t *testing.T, cache *StorageCache, out primitives.TxOut) {

	id := TxOutId{TxHash: *out.TxHash, Nout: out.Nout}
	defaultTxOut := TxOutData{Addr: "", Value: 0}
	
	data, err := cache.GetTxOut(id)
	if err != nil {
		t.Errorf("GetTxOut(%v): %v", id, err)
		return
	}

	if data != defaultTxOut {
		t.Error("Should have returned default value not", data)
	}

	if cont, _ := cache.Contains(id); cont {
		t.Error("Shouldn't be in storage: ",  out)
	}
}

// Test cache length
func cacheLen(t *testing.T, cache *StorageCache, size int) {
	l, err := cache.Len()
	if err != nil {
		t.Errorf("Len(): %v", err)
		return
	}
	if l != size {
		t.Errorf("Len(): Expecting %v returned %v", size, l)
		return
	}
}

// Test cache uncommited length
func cacheUncommittedLen(t *testing.T, cache *StorageCache, size int) {	
	l := cache.UncommittedLen()
	if l != size {
		t.Errorf("UncommittedLen(): Expecting %v returned %v", size, l)
		return
	}

}

// Test cache Len and UncommitedLen
func TestCacheLen(t *testing.T) {

	storage, _ := NewSQLiteStorage(":memory:")
	cache, _ := NewStorageCache(storage, 10000)
	cache.SetHeight(100)
	cache.SetHash(primitives.MainNetGenesisHash)

	cacheLen(t, cache, 0)
	cacheUncommittedLen(t, cache, 0)

	// Add some txout and check lengths
	outs   := mockTxOuts(1000, 2000, 1, 0)
	outsId := TxOutToId(outs)

	for _, out := range outs {
		cache.AddTxOut(out)
	}
	cacheLen(t, cache, 0)
	cacheUncommittedLen(t, cache, len(outs))

	// Commit changes and check again
	cache.Commit()
	cacheLen(t, cache, len(outs))
	cacheUncommittedLen(t, cache, 0)

	// Delete some of the TxOuts
	for _, out := range outsId[:500] {
		cache.DelTxOut(out)
	}

	cacheLen(t, cache, len(outs))
	cacheUncommittedLen(t, cache, 500)
	
	cache.Commit()

	cacheLen(t, cache, len(outs) - 500)
	cacheUncommittedLen(t, cache, 0)

	// Delete some uncommitted TxOuts
	storage, _ = NewSQLiteStorage(":memory:")
	cache, _ = NewStorageCache(storage, 10000)
	cache.SetHeight(100)
	
	more   := mockTxOuts(4000, 5000, 1, 0)
	moreId := TxOutToId(more)
	
	for _, out := range more {
		cache.AddTxOut(out)
	}
	cacheLen(t, cache, 0)
	cacheUncommittedLen(t, cache, len(more))

	for _, out := range moreId[:500] {
		cache.DelTxOut(out)
	}
	cacheLen(t, cache, 0)
	cacheUncommittedLen(t, cache, len(more)-500)

	// Mixed add and delete
	storage, _ = NewSQLiteStorage(":memory:")
	cache, _ = NewStorageCache(storage, 10000)
	cache.SetHeight(100)
	
	for _, out := range more {
		cache.AddTxOut(out)
	}
	for _, out := range outsId {
		cache.DelTxOut(out)
	}
	cacheLen(t, cache, 0)
	cacheUncommittedLen(t, cache, len(more)+len(outs))

	cache.Commit()

	cacheLen(t, cache, len(more))
	cacheUncommittedLen(t, cache, 0)

	// Read initial length from storage	
	storage, _ = NewSQLiteStorage(":memory:")
	for _, out := range outs {
		storage.Set(out)
	}
	
	cache, _ = NewStorageCache(storage, 10000)
	cache.SetHeight(100)
	
	cacheLen(t, cache, len(outs))
	cacheUncommittedLen(t, cache, 0)
}


// Test SetHash and GetHash methods
func TestCacheGetHashHeight(t *testing.T) {
	
	// Test height and hash are loaded from uninitialized storage.
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}

	cache, err := NewStorageCache(storage, 10000)
	if err != nil {
		t.Error("NewStorageCache(): ", err)
		return
	}

	if height := cache.GetHeight(); height != -1 {
		t.Errorf("GetHeight(): Expected -1 returned %v", height) 
		return
	}
	if hash := cache.GetHash(); hash != primitives.ZeroHash {
		t.Errorf("GetHash(): expected %v, returned %v", 
			primitives.ZeroHash, hash)
		return
	}

	// Test height loaded from initialized storage
	storage, err = NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	
	storage.SetLastBlock(999, primitives.MainNetGenesisHash)

	cache, err = NewStorageCache(storage, 10000)
	if err != nil {
		t.Error("NewStorageCache(): ", err)
		return
	}
	
	if height := cache.GetHeight(); height != 999 {
		t.Errorf("GetHeight(): Expected 999 returned %v", height) 
		return
	}
	if hash := cache.GetHash(); hash != primitives.MainNetGenesisHash {
		t.Errorf("GetHash(): expected %v, returned %v", 
			primitives.MainNetGenesisHash, hash)
		return
	}
	cache.Commit()

	// Set height and hash and check before and after commit
	cache.SetHeight(2000)
	cache.SetHash(primitives.ZeroHash)
	if height := cache.GetHeight(); height != 2000 {
		t.Errorf("GetHeight(): Expected 2000 returned %v", height)
		return
	}	
	if hash := cache.GetHash(); hash != primitives.ZeroHash {
		t.Errorf("GetHash(): expected %v, returned %v", 
			primitives.ZeroHash, hash)
		return
	}


	err = cache.Commit()
	if err != nil {
		t.Error("Commit(): ", err)
	}

	if height := cache.GetHeight(); height != 2000 {
		t.Errorf("GetHeight(): Expected 2000 returned %v", height)
		return
	}
	if hash := cache.GetHash(); hash != primitives.ZeroHash {
		t.Errorf("GetHash(): expected %v, returned %v", 
			primitives.ZeroHash, hash)
		return
	}

	// Check new height was commited to storage
	height, hash, err := storage.GetLastBlock()
	if err != nil {
		t.Errorf("Unable to get last block from storage", err)
	}
	if height != 2000 {
		t.Errorf("Commit() height failure expected 2000 stored %v", height)
		return
	}
	if hash != primitives.ZeroHash {
		t.Errorf("Commit() hash failure expected %v stored %v", primitives.ZeroHash, hash)
		return
	}
}



// Test Contains
func TestCacheContains(t *testing.T) {
	storage, _ := NewSQLiteStorage(":memory:")
	cache, _ := NewStorageCache(storage, 10000)
	cache.SetHeight(100)

	checkContains := func (sto *StorageCache, out primitives.TxOut) {
		id := TxOutId{TxHash: *out.TxHash, Nout: out.Nout}
		con, err := cache.Contains(id)
		if  err != nil {
			t.Error("Contains(): ", err)
			return 
		}
		if !con {
			t.Error("Contains(): Didn't contain", id)
		}
		return
	}
	
	checkNotContains := func (sto *StorageCache, out primitives.TxOut) {
		id := TxOutId{TxHash: *out.TxHash, Nout: out.Nout}
		con, err := cache.Contains(id)
		if  err != nil {
			t.Error("Contains(): ", err)
			return 
		}
		if con {
			t.Error("Contains(): Didn't contain", id)
		}
		return 
	}

	// Add some txouts
	outs   := mockTxOuts(1000, 2000, 1, 0)
	outsId := TxOutToId(outs)

	for _, out := range outs {
		cache.AddTxOut(out)
	}
	for _, out := range outs {
		checkContains(cache, out)
	}

	// Commit the additions
	cache.Commit()
	for _, out := range outs {
		checkContains(cache, out)
	}

	// Delete some of the TxOuts
	for _, out := range outsId[:500] {
		cache.DelTxOut(out)
	}
	for _, out := range outs[:500] {
		checkNotContains(cache, out)
	}
	for _, out := range outs[500:] {
		checkContains(cache, out)
	}

	// Commit deletions
	cache.Commit()	
	for _, out := range outs[:500] {
		checkNotContains(cache, out)
	}
	for _, out := range outs[500:] {
		checkContains(cache, out)
	}
}

// Test Resize method
func TestCacheResize(t *testing.T) {
	storage, _ := NewSQLiteStorage(":memory:")
	cache, _ := NewStorageCache(storage, 10000)

	cache.Resize(10)
}

// Test GetTxOut
func TestCacheGetTxOut(t *testing.T) {	

	// Initialize storage with some mock TxOuts
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	
	storedOuts := mockTxOuts(10000, 50000, 2, 0)
	initStorage(t, storage, storedOuts)


	// Start cache
	cache, err := NewStorageCache(storage, 10000)
	if err != nil {
		t.Error("NewStorageCache(): ", err)
		return
	}

	// Use GetTxOut to check storage initial data
	for _, out := range storedOuts {
		cacheContains(t, cache, out)
	}

	// Add and Get new txouts
	moreOuts := mockTxOuts(200000, 201000, 2, 0)

	for _, out := range moreOuts {
		cache.AddTxOut(out)
	}

	for _, out := range moreOuts {
		cacheContains(t, cache, out)
	}

	// Commit and Get again
	cache.SetHeight(100)
	if err := cache.Commit(); err != nil {
		t.Error("Commit(): ", err)
		return
	}

	for _, out := range moreOuts {
		cacheContains(t, cache, out)
	}
}

// Test BulkGetTxOut
func TestCacheBulkGetTxOut(t *testing.T) {
	// Initialize storage with some mock TxOuts
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	
	stored     := mockTxOuts(10000, 10200, 2, 0)
	storedId   := TxOutToId(stored)
	storedData := TxOutToData(stored)
	initStorage(t, storage, stored)


	// Start cache
	cache, err := NewStorageCache(storage, 10000)
	if err != nil {
		t.Error("NewStorageCache(): ", err)
		return
	}

	// Use BulkGetTxOut to check storage initial data
	for i:=0; i < len(stored); i++ {
		returned, err := cache.BulkGetTxOut(storedId[:i])
		if err != nil {
			t.Error("BulkGetTxOut(): ", err)
		}
		if len(returned) != len(storedData[:i]) {
			t.Error("BulkGetTxOut(): Unexpected lenght")
			return
		}
		for n, data := range storedData[:i] {
			if returned[n] != data {
				t.Error("BulkGetTxOut(): Expected %v returned %v", data, returned[n])
				return
			}
		}
	}

	// BulkGet TxOuts half committed and half not
	mixed     := mockTxOuts(200000, 201000, 2, 0)
	mixedIds  := TxOutToId(mixed)
	mixedData := TxOutToData(mixed)
	for _, out := range mixed[:500] {
		cache.AddTxOut(out)
	}
	
	cache.Commit()
	
	for _, out := range mixed[500:] {
		cache.AddTxOut(out)
	}

	returned , err := cache.BulkGetTxOut(mixedIds)
	if err != nil {
		t.Error("BulkGetTxOut(): ", err)
		return
	}

	for n, data := range mixedData {
		if returned[n] != data {
			t.Error("BulkGetTxOut(): Expected %v returned %v", data, returned[n])
			return
		}
	}
}

// Test BulkGetTxOut with duplicated TxOutIds
func TestCacheBulkGetTxOutDuplicates(t *testing.T) {
	
	txout1 := mockTxOuts(10000, 10200, 2, 0)
	
	// Initialize storage and cache with some mock TxOuts
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error("NewSQLiteStorage(): ", err)
		return
	}
	initStorage(t, storage, txout1)

	cache, err := NewStorageCache(storage, 10000)
	if err != nil {
		t.Error("NewStorageCache(): ", err)
		return
	}

	// Create duplicated id slice
	txout2 := mockTxOuts(10000, 10200, 2, 0)
	duplicates     := append(txout1, txout2...)
	duplicatesId   := TxOutToId(duplicates)
	duplicatesData := TxOutToData(duplicates)

	returned, err := cache.BulkGetTxOut(duplicatesId)
	if err != nil {
		t.Error("BulkGetTxOut(): ", err)
		return
	}

	for n, out := range duplicatesData {
		if out != returned[n] {
			t.Errorf("BulkGetTxOut(): Expecting %v returned %v", out, returned[n])
			return
		}
	}
}

// Test AddTxOut
func TestCacheAddTxOut(t *testing.T) {
	storage, _ := NewSQLiteStorage(":memory:")
	cache, _   := NewStorageCache(storage, 10000)
	cache.SetHeight(1000)
	cache.SetHash(primitives.MainNetGenesisHash)

	// Add TxOut and check they are available before commit
	outs   := mockTxOuts(1000, 2000, 2, 0)
	for _, out := range outs {
		cache.AddTxOut(out)
	}

	for _, out := range outs {
		cacheContains(t, cache, out)
	}

	// Commit and check again
	cache.Commit()
	for _, out := range outs {
		cacheContains(t, cache, out)
	}

	// Delete some outputs and check
	outsId := TxOutToId(outs)
	for _, del := range outsId[:500] {
		cache.DelTxOut(del)
	}

	for _, del := range outs[:500] {
		cacheNotContains(t, cache, del)
	}

	for _, out := range outs[500:] {
		cacheContains(t, cache, out)
	}

	// Commit deletions and check
	cache.Commit()

	for _, del := range outs[:500] {
		cacheNotContains(t, cache, del)
	}

	for _, out := range outs[500:] {
		cacheContains(t, cache, out)
	}

	// Check height and hash was committed to storage
	height, hash, _ := storage.GetLastBlock()
	if height != 1000 {
		t.Error("Commit(): Height wasn't committed to storage")
	}
	if hash != primitives.MainNetGenesisHash {
		t.Error("Commit(): Hash wasn't committed to storage")
	}
}

// Test DelTxOut
func TestCacheDelTxOut(t *testing.T) {
	storage, _ := NewSQLiteStorage(":memory:")
	cache, _   := NewStorageCache(storage, 10000)

	outs   := mockTxOuts(0, 400, 1, 1)
	outsId := TxOutToId(outs)

	for _, out := range outs {
		cache.AddTxOut(out)
	}

	// Remove half of the inserted txouts
	for _, out := range outsId[:100] {
		cache.DelTxOut(out)
	}

	for _, out := range outs[:100] {
		cacheNotContains(t, cache, out)
	}

	for _, out := range outs[100:] {
		cacheContains(t, cache, out)
	}

	// Remove commited  txout
	cache.SetHeight(1000)
	if err := cache.Commit(); err != nil {
		t.Error("Commit(): ", err)
		return
	}

	for _, out := range outsId[100:200] {
		cache.DelTxOut(out)
	}

	for _, out := range outs[:200] {
		cacheNotContains(t, cache, out)
	}
	for _, out := range outs[200:] {
		cacheContains(t, cache, out)
	}
	
	if err := cache.Commit(); err != nil {
		t.Error("Commit(): ", err)
		return
	}

	// Remove non existent txout
	for _, out := range outsId[:100] {
		cache.DelTxOut(out)
	}

	if err := cache.Commit(); err != nil {
		t.Error("Commit(): ", err)
		return
	}

	for _, out := range outs[:200] {
		cacheNotContains(t, cache, out)
	}
	for _, out := range outs[200:] {
		cacheContains(t, cache, out)
	}
}

// Test Commit errors
func TestCacheCommitErrors(t *testing.T) {

	// Invalid TxOuts
	hash := mockHash(10)
	negativeTxOut  := primitives.NewTxOut(&hash, 1, "fake_address", -10)
	zeroTxOut      := primitives.NewTxOut(&hash, 1, "fake_address", 0)
	noAddressTxOut := primitives.NewTxOut(&hash, 1, "", 10)
	validTxOut     := primitives.NewTxOut(&hash, 1, "fake_address", 10)

	// Check negative value TxOut
	storage, _ := NewSQLiteStorage(":memory:")
	cache, _   := NewStorageCache(storage, 10000)
	cache.SetHeight(1000)

	cache.AddTxOut(*negativeTxOut)
	if err := cache.Commit(); err != ErrNegativeUtxo {
		t.Error(err)
		return
	}

	// Check zero value TxOut
	storage, _ = NewSQLiteStorage(":memory:")
	cache, _   = NewStorageCache(storage, 10000)
	cache.SetHeight(1000)

	cache.AddTxOut(*zeroTxOut)
	if err := cache.Commit(); err != ErrUnexpendableUtxo {
		t.Error(err)
		return
	}

	// Check address-less TxOut
	storage, _ = NewSQLiteStorage(":memory:")
	cache, _   = NewStorageCache(storage, 10000)
	cache.SetHeight(1000)

	cache.AddTxOut(*noAddressTxOut)
	if err := cache.Commit(); err != ErrUnexpendableUtxo {
		t.Error(err)
		return
	}

	// Check negative height
	storage, _ = NewSQLiteStorage(":memory:")
	cache, _   = NewStorageCache(storage, 10000)
	cache.SetHeight(-1000)

	cache.AddTxOut(*validTxOut)
	if err := cache.Commit(); err != ErrNegativeHeight {
		t.Error(err)
		return
	}

	// Check there's a rollback on error
	storage, _ = NewSQLiteStorage(":memory:")
	cache, _   = NewStorageCache(storage, 10000)
	cache.SetHeight(1000)

	outs   := mockTxOuts(0, 400, 1, 1)
	for _, out := range outs {
		cache.AddTxOut(out)
	}
	cache.AddTxOut(*negativeTxOut)

	if err := cache.Commit(); err != ErrNegativeUtxo {
		t.Error(err)
		return
	}

	// Verify nothing was written to storage
	if l, err := storage.Len(); err != nil || l != 0 {
		t.Error("Something was written to storage with a failed commit")
		return
	}

	height, hash, err := storage.GetLastBlock()
	if err != nil || height != -1 || hash != primitives.ZeroHash {
		t.Error("New height was written storage with a failed commit")
		return
	}
}

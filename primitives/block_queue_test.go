package primitives

import (
	"fmt"
	"testing"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

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


func initBlocks() (*Block, *Block){

	hash1 := mockHash(1)
	hash2 := mockHash(2)
	addr1 := mockAddress(1)
	addr2 := mockAddress(2)

	tx1 := NewTx(&hash1)
	tx2 := NewTx(&hash2)

	txOut1 := NewTxOut(&hash1, 0, addr1, 10)

	txIn2  := NewTxOut(&hash2, 0, addr1, 10)
	txOut2 := NewTxOut(&hash2, 0, addr2, 5)

	tx1.AddOut(txOut1)

	tx2.AddIn(txIn2)
	tx2.AddOut(txOut2)

	block1 := NewBlock(ZeroHash, ZeroHash, 1)
	block2 := NewBlock(MainNetGenesisHash, ZeroHash, 2)

	block1.AddTx(tx1)
	block2.AddTx(tx2)

	return block1, block2
}


// Check basic queue operations
func TestQueuePopFront(t *testing.T) {

	block1, block2 := initBlocks()
	queue := NewBlockQueue()
	
	// Add both blocks at the end of the queue
	////////////////////////////////////////////
	queue.PushBack(block1)
	queue.PushBack(block2)
	if l := queue.Len(); l != 2 {
		t.Errorf("Len(): expecting 2 returned %v", l)
	}

	// pop 
	block := queue.PopFront()
	if block != block1 {
		t.Errorf("PopFront(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}
	
	block = queue.PopFront()
	if block != block2 {
		t.Errorf("PopFront(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}

	// Pop from an empty queue
	block = queue.PopFront()
	if block != nil {
		t.Error("PopFront(): Queue wasn't empty")
	}
	if l := queue.Len(); l != 0 {
		t.Errorf("Len(): expecting 0 returned %v", l)
	}

	// Add one block at the end and one at the beginning
	////////////////////////////////////////////////////
	queue.PushBack(block1)
	queue.PushFront(block2)	
	if l := queue.Len(); l != 2 {
		t.Errorf("Len(): expecting 2 returned %v", l)
	}

	// pop 
	block = queue.PopFront()
	if block != block2 {
		t.Errorf("PopFront(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}
	
	block = queue.PopFront()
	if block != block1 {
		t.Errorf("PopFront(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}

	// Pop from an empty queue
	block = queue.PopFront()
	if block != nil {
		t.Error("PopFront(): Queue wasn't empty")
	}
	if l := queue.Len(); l != 0 {
		t.Errorf("Len(): expecting 0 returned %v", l)
	}
}


// Check basic queue operations
func TestQueuePopBack(t *testing.T) {

	block1, block2 := initBlocks()
	queue := NewBlockQueue()
	
	// Add both blocks at the end of the queue
	////////////////////////////////////////////
	queue.PushBack(block1)
	queue.PushBack(block2)
	if l := queue.Len(); l != 2 {
		t.Errorf("Len(): expecting 2 returned %v", l)
	}

	// pop 
	block := queue.PopBack()
	if block != block2 {
		t.Errorf("PopBack(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}
	
	block = queue.PopFront()
	if block != block1 {
		t.Errorf("PopBack(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}

	// Pop from an empty queue
	block = queue.PopFront()
	if block != nil {
		t.Error("PopBack(): Queue wasn't empty")
	}
	if l := queue.Len(); l != 0 {
		t.Errorf("Len(): expecting 0 returned %v", l)
	}

	// Add one block at the end and one at the beginning
	////////////////////////////////////////////////////
	queue.PushBack(block1)
	queue.PushFront(block2)	
	if l := queue.Len(); l != 2 {
		t.Errorf("Len(): expecting 2 returned %v", l)
	}

	// pop 
	block = queue.PopBack()
	if block != block1 {
		t.Errorf("PopBack(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}
	
	block = queue.PopBack()
	if block != block2 {
		t.Errorf("PopBack(): Expecting %v not %v", block1.Hash, block.Hash)
		return
	}

	// Pop from an empty queue
	block = queue.PopBack()
	if block != nil {
		t.Error("PopBack(): Queue wasn't empty")
	}
	if l := queue.Len(); l != 0 {
		t.Errorf("Len(): expecting 0 returned %v", l)
	}
}




func TestQueueBalance(t *testing.T) {
	
	block1, block2 := initBlocks()
	addr1 := block1.Transactions[0].Out[0].Addr
	addr2 := block2.Transactions[0].Out[0].Addr
	queue := NewBlockQueue()

	// Add first block and check balance
	queue.PushBack(block1)
	balance, _ := queue.GetBalance(addr1)
	expected := int64(10)
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr1, expected, balance)
		return
	}

	// Add second block and check balance again
	queue.PushBack(block2)
	balance, _ = queue.GetBalance(addr1)
	expected = 0
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr1, expected, balance)
		return
	}
	
	balance, _ = queue.GetBalance(addr2)
	expected = 5
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr2, expected, balance)
		return
	}

	// Remove oldest block and check balance
	queue.PopFront()
	
	balance, _ = queue.GetBalance(addr1)
	expected = -10
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr1, expected, balance)
		return
	}
	
	balance, _ = queue.GetBalance(addr2)
	expected = 5
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr2, expected, balance)
		return
	}

	// Remove remaining block and check balance again
	queue.PopBack()
	
	balance, _ = queue.GetBalance(addr1)
	expected = 0
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr1, expected, balance)
		return
	}
	
	balance, _ = queue.GetBalance(addr2)
	expected = 0
	if balance != expected {
		t.Errorf("GetBalance(%v): Expectiong %v returned %v", addr2, expected, balance)
		return
	}
}

func TestQueueTxIndex(t *testing.T) {

	block1, block2 := initBlocks()
	hash1 := block1.Transactions[0].Hash
	hash2 := block2.Transactions[0].Hash
	queue := NewBlockQueue()

	// Add blocks and check transactions
	queue.PushBack(block1)
	tx1 := queue.Tx(*hash1)
	tx2 := queue.Tx(*hash2)
	if tx1 == nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash1)
		return
	}
	if tx2 != nil {
		t.Errorf("Tx(%v): Transaction shouldn't be indexed", *hash2)
		return
	}

	queue.PushFront(block2)
	tx1 = queue.Tx(*hash1)
	tx2 = queue.Tx(*hash2)
	if tx1 == nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash1)
		return
	}
	if tx2 == nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash2)
		return
	}

	// Remove blocks and check transactions
	queue.PopBack()
	tx1 = queue.Tx(*hash1)
	tx2 = queue.Tx(*hash2)
	if tx1 != nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash1)
		return
	}
	if tx2 == nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash2)
		return
	}


	queue.PopFront()
	tx1 = queue.Tx(*hash1)
	tx2 = queue.Tx(*hash2)		
	if tx1 != nil {
		t.Errorf("Tx(%v): Transaction shouldn't be indexed", *hash1)
		return
	}
	if tx2 != nil {
		t.Errorf("Tx(%v): Transaction shouldn't be indexed", *hash2)
		return
	}
}

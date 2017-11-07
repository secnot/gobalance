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
	tx1, _ := queue.Tx(*hash1)
	tx2, _ := queue.Tx(*hash2)
	if tx1 == nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash1)
		return
	}
	if tx2 != nil {
		t.Errorf("Tx(%v): Transaction shouldn't be indexed", *hash2)
		return
	}

	queue.PushFront(block2)
	tx1, _ = queue.Tx(*hash1)
	tx2, _ = queue.Tx(*hash2)
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
	tx1, _ = queue.Tx(*hash1)
	tx2, _ = queue.Tx(*hash2)
	if tx1 != nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash1)
		return
	}
	if tx2 == nil {
		t.Errorf("Tx(%v): Transaction wasn't indexed", *hash2)
		return
	}


	queue.PopFront()
	tx1, _ = queue.Tx(*hash1)
	tx2, _ = queue.Tx(*hash2)		
	if tx1 != nil {
		t.Errorf("Tx(%v): Transaction shouldn't be indexed", *hash1)
		return
	}
	if tx2 != nil {
		t.Errorf("Tx(%v): Transaction shouldn't be indexed", *hash2)
		return
	}
}

// Test transactions are only indexed once when an address appears more than once in
// the same transaction.
func TestQueueTxIndexOnce(t *testing.T) {

	tx1Hash := mockHash(1)
	tx2Hash := mockHash(2)
	tx3Hash := mockHash(3)

	addr1 := mockAddress(1)
	addr2 := mockAddress(2)
	addr3 := mockAddress(3)


	// Tx1
	tx1 := NewTx(&tx1Hash)
	
	tx1Out1 := NewTxOut(&tx1Hash, 0, addr1, 10)
	tx1Out2 := NewTxOut(&tx1Hash, 1, addr1, 20)
	tx1Out3 := NewTxOut(&tx1Hash, 2, addr2, 10)
	tx1.AddOut(tx1Out1)
	tx1.AddOut(tx1Out2)
	tx1.AddOut(tx1Out3)

	tx1In := NewTxOut(&tx3Hash, 0, addr1, 5)
	tx1.AddIn(tx1In)

	// Tx2
	tx2 := NewTx(&tx2Hash)
	tx2Out := NewTxOut(&tx2Hash, 0, addr2, 10)
	tx2.AddOut(tx2Out)

	tx2In  := NewTxOut(&tx3Hash, 0, addr3, 5)
	tx2.AddIn(tx2In)
	
	// Block
	block := NewBlock(MainNetGenesisHash, ZeroHash, 2)

	block.AddTx(tx1)
	block.AddTx(tx2)

	// Populate queue
	queue := NewBlockQueue()
	queue.PushBack(block)
	
	// Check addr1
	addr1Tx := queue.GetTx(addr1)
	if len(addr1Tx) != 1 {
		t.Errorf("queue.GetTx(): Expencting 1 transaction, returned %v", len(addr1Tx))
	}

	if addr1Tx[0] != tx1 {
		t.Errorf("queue.GetTx()[0]: Unexpected transaction")
	}

	// Check addr2
	addr2Tx := queue.GetTx(addr2)
	if len(addr2Tx) != 2 {
		t.Errorf("queue.GetTx(): Expencting 2 transaction, returned %v", len(addr2Tx))
	}
	
	if addr2Tx[0] != tx1 {
		t.Errorf("queue.GetTx()[0]: Unexpected transaction")
	}
	if addr2Tx[1] != tx2 {
		t.Errorf("queue.GetTx()[1]: Unexpected transaction")
	}

	// Check balance
	if balance, ok := queue.GetBalance(addr1); balance != 25 || ok != true {
		t.Errorf("queue.GetBalance(%v): expecting 25 returned %v", addr1, balance)
	}

	if balance, ok := queue.GetBalance(addr2); balance != 20 || ok != true {
		t.Errorf("queue.GetBalance(%v): expecting 20 returned %v", addr1, balance)
	}
	
	if balance, ok := queue.GetBalance(addr3); balance != -5 || ok != true{
		t.Errorf("queue.GetBalance(%v): expecting -5 returned %v", addr1, balance)
	}


	// Remove block and check again
	retBlock := queue.PopFront()
	if retBlock != block {
		t.Error("queue.PopFront(): returned unexpected block")
	}	
	
	if len(queue.GetTx(addr2)) != 0 {
		t.Errorf("queue.GetTx(): Expencting 2 transaction, returned %v", len(addr2Tx))
	}
	
	if len(queue.GetTx(addr1)) != 0 {
		t.Errorf("queue.GetTx(): Expencting 1 transaction, returned %v", len(addr1Tx))
	}
	
	if balance, ok := queue.GetBalance(addr1); balance != 0 || ok != false{
		t.Errorf("queue.GetBalance(%v): expecting 0 returned %v", addr1, balance)
	}

	if balance, ok := queue.GetBalance(addr2); balance != 0 || ok != false {
		t.Errorf("queue.GetBalance(%v): expecting 0 returned %v", addr1, balance)
	}
	
	if balance, ok := queue.GetBalance(addr3); balance != 0 || ok != false{
		t.Errorf("queue.GetBalance(%v): expecting 0 returned %v", addr1, balance)
	}
}

func TestQueueBack(t *testing.T) {
	// Test an empty queue returns nil
	queue := NewBlockQueue()
	if queue.Back() != nil {
		t.Error("queue.Back(): Should have returned nil")
	}

	block1, block2 := initBlocks()
	queue.PushFront(block1)
	if queue.Back() != block1 {
		t.Error("queue.Back(): returned unexpected block")
	}
	
	queue.PushFront(block2)
	if queue.Back() != block1 {
		t.Error("queue.Back(): returned unexpected block")
	}
}


func testQueueFront(t *testing.T) {
	// Test an empty queue returns nil
	queue := NewBlockQueue()
	if queue.Front() != nil {
		t.Error("queue.Front(): Should have returned nil")
	}

	block1, block2 := initBlocks()
	queue.PushBack(block1)
	if queue.Front() != block1 {
		t.Error("queue.Back(): returned unexpected block")
	}
	
	queue.PushBack(block2)
	if queue.Front() != block1 {
		t.Error("queue.Back(): returned unexpected block")
	}

	queue.PopFront()
	if queue.Front() != block2 {
		t.Error("queue.Back(): returned unexpected block")
	}
}



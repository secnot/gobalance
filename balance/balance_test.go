package balance

import (
	"fmt"
	"net"
	"time"
	"testing"
	"github.com/secnot/gobalance/recent_tx"
	"github.com/secnot/gobalance/height"
	"github.com/secnot/gobalance/api"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/interfaces"
	"github.com/secnot/gobalance/interfaces/mocks"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)


// Helper
////////////////////////////////////////

// BalanceIsEqual checks the balance of the address in expected map
func BalanceIsEqual(t *testing.T, cache *BalanceCache, ip net.IP, expected map[string]int64) {

	for address, expectedBalance := range expected {
		cachedBalance, err := cache.GetBalance(address, ip)
		if err != nil {
			t.Error(err)
			return
		}
		if cachedBalance != expectedBalance {
			t.Errorf("GetBalanced(%v): expecting %v returned %v\n", address, expectedBalance, cachedBalance)
			return
		}
	}
}

// Create mock hash
func mockHash(id uint) chainhash.Hash {
	hashStr := fmt.Sprintf("txhash_%v", id)
	var hash chainhash.Hash
	copy(hash[:], hashStr)
	return hash
}

// buildBlock creates a block with a single transaction that moves bitcoins 
// from source to destination addresses
func initBlock(source map[string]int64, destination map[string]int64) *primitives.Block {

	hash1 := mockHash(1)
	hash2 := mockHash(2)
	hash3 := mockHash(3)

	tx := primitives.NewTx(&hash1)

	// add inputs
	inNum := uint32(0)
	for addr, value := range source {
		in := primitives.NewTxOut(&hash2, inNum, addr, value)
		tx.AddIn(in)
		inNum++
	}

	// add outputs
	outNum := uint32(0)
	for addr, value := range destination {
		out := primitives.NewTxOut(&hash3, outNum, addr, value)
		tx.AddOut(out)
		outNum++
	}

	// Create block and add single transaction
	block := primitives.NewBlock(primitives.MainNetGenesisHash, primitives.ZeroHash, 2)
	block.AddTx(tx)

	return block
}




// Tests
////////////////////////////

// Test balance operation
func TestBalance(t *testing.T) {
	balance := map[string]int64{
		"address1": 12,
		"address2": 50 ,
	}

	blockM := mocks.NewBlockMock(balance, 1000, true)
	peerM  := mocks.NewPeerMock("localhost:9090")
	cache_size := 10
	balanceCache := NewBalanceCache(blockM, peerM, cache_size)

	localhost := net.ParseIP("127.0.0.1")

	// Test initial balance
	balance["unknow_address"] = 0
	BalanceIsEqual(t, balanceCache, localhost, balance)

	// Send block update and check address balance changed
	src := map[string]int64{"address1": 11}
	dst := map[string]int64{"address3": 9, "address2": 1}
	block := initBlock(src, dst)
	update := interfaces.NewBlockUpdate(interfaces.OP_NEWBLOCK, block)	
	blockM.SendBlockUpdate(update)

	// "address3" returns 0 because the balance isn't cached so it is retrieved from 
	// the mock block manager that allways return 0
	expected := map[string]int64{"address1": 1, "address2": 51, "address3": 0}
	time.Sleep(100*time.Millisecond)
	BalanceIsEqual(t, balanceCache, localhost, expected)

	// Cleanup
	blockM.Stop()
	peerM.Stop()
	balanceCache.Stop()
}


// TestRemoteBalance test balance is requested from remote peer when 
// committing.
func TestRemoteBalance(t *testing.T) {

	// Start http api for remote peer
	//////////////////////////////////
	remoteBalance := map[string]int64{
		"address1": 12,
		"address2": 50,
		"address3": 90,
	}
	remoteHeight := int64(1000)
	cache_size := 10
	localhost := net.ParseIP("127.0.0.1")

	remoteBlockM    := mocks.NewBlockMock(remoteBalance, remoteHeight, true)
	remotePeerM     := mocks.NewPeerMock("localhost:9090")
	remoteHeightM   := height.NewHeightCache(remoteBlockM)
	remoteRecentTxM := recent_tx.NewRecentTxCache(remoteBlockM, 10)	
	remoteBalanceCache := NewBalanceCache(remoteBlockM, remotePeerM, cache_size)

	remoteServer := api.StartApi("127.0.0.1:9999", "/", remoteBalanceCache, remoteRecentTxM, remoteHeightM)
	

	// Start local peer
	////////////////////
	localBalance := map[string]int64 {
		"address1": 1012,
		"address2": 1050,
	}
	localHeight := int64(1000)

	localBlockM  := mocks.NewBlockMock(localBalance, localHeight, true)
	localPeerM   := mocks.NewPeerMock("127.0.0.1:9999")
	localBalanceCache := NewBalanceCache(localBlockM, localPeerM, cache_size)


	// Check balance before commit
	beforeBalance := localBalance
	BalanceIsEqual(t, localBalanceCache, localhost, beforeBalance)
	
	// Check balance during commit
	update := interfaces.NewBlockUpdate(interfaces.OP_COMMIT, nil)
	localBlockM.SendBlockUpdate(update)
	time.Sleep(500*time.Millisecond)
	BalanceIsEqual(t, localBalanceCache, localhost, remoteBalance) 

	// Check balance unavailable remote peer
	localPeerM.SetPeer("127.0.0.1:55555")
	time.Sleep(100*time.Millisecond)	
	bal, err := localBalanceCache.GetBalance("address1", localhost)
	if err == nil {
		t.Errorf("GetBalance(): Expecting error while requesting from unavailable peer, returned %v", bal)
		return
	}

	// Check balance after commit	
	update = interfaces.NewBlockUpdate(interfaces.OP_COMMIT_DONE, nil)
	localBlockM.SendBlockUpdate(update)
	time.Sleep(500*time.Millisecond)
	BalanceIsEqual(t, localBalanceCache, localhost, localBalance) 

	//remoteServer.Shutdown()
	remoteBlockM.Stop()
	remotePeerM.Stop()
	remoteHeightM.Stop()
	remoteRecentTxM.Stop()
	remoteBalanceCache.Stop()
	remoteServer.Stop()

	localPeerM.Stop()
	localBlockM.Stop()
	localBalanceCache.Stop()
	
	// Wait all is closed
	time.Sleep(time.Second)
}

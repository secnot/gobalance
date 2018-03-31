package height

import (
	"testing"
	"time"
	"github.com/secnot/gobalance/interfaces"
	"github.com/secnot/gobalance/interfaces/mocks"
	"github.com/secnot/gobalance/primitives"
)


// mockBlock
func mockBlock(height uint64)  *primitives.Block {
	block := primitives.NewBlock(primitives.MainNetGenesisHash, primitives.ZeroHash, height)
	return block
}


func TestHeight(t *testing.T) {
	initialBalance := map[string]int64{
		"address1": 12,
		"address2": 50,
		"address3": 90,
	}
	initialHeight := int64(1000)

	blockM := mocks.NewBlockMock(initialBalance, initialHeight, true)
	
	// Test initial height retrieved from block manager
	heightCache := NewHeightCache(blockM)
	time.Sleep(100*time.Millisecond)
	if h := heightCache.GetHeight(); h != uint64(initialHeight)  {
		t.Errorf("GetHeight(): Unexpected intial height %v\n", h)
	}

	// Test height update with new block
	block := mockBlock(9999)
	update := interfaces.NewBlockUpdate(interfaces.OP_NEWBLOCK, block)	
	blockM.SendBlockUpdate(update)
	time.Sleep(400*time.Millisecond)
	if h := heightCache.GetHeight(); h != 9999 {
		t.Errorf("GetHeight(): Height was not updated after receiving a new block %v\n", h)
	}

	// Test height update after backtrack
	// TODO: for now backtracking just usbstracts one from the current height
	// it would be better to keep a queue with the previous heights
	update = interfaces.NewBlockUpdate(interfaces.OP_BACKTRACK, nil)
	blockM.SendBlockUpdate(update)
	time.Sleep(400*time.Millisecond)
	if h := heightCache.GetHeight(); h != 9998 {
		t.Errorf("GetHeight(): Unexpected height after backtracks %v\n", h)
	}
}


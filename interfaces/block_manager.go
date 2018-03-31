package interfaces

import (
	"github.com/secnot/gobalance/primitives"
)

// Subscriber updates types
type UpdateClass int

const (
	// Signal a new block in the chain (block included in msg)
	OP_NEWBLOCK  UpdateClass = iota

	// Signal last block is not longer part of the longest chain and was
	// discarded. (block included in msg)
	OP_BACKTRACK
	
	// Signal that a commit will start soon, once the commit has started
	// the manager will be unresponsive until finished.
	OP_COMMIT
	
	// Signal that the commit has finished
	OP_COMMIT_DONE
)

// Struct used to send chain updates to subscribers
type BlockUpdate struct {
	Class UpdateClass
	Block *primitives.Block
}

func NewBlockUpdate(class UpdateClass, block *primitives.Block) BlockUpdate{
	return BlockUpdate{
		Class: class,
		Block: block,
	}
}

// Channel type used to send subscriber updates
type UpdateChan chan BlockUpdate


// Block manager interface only purpose is to allow mock testing
type BlockManager interface {
	// Subscribe to new block updates
	Subscribe(chanSize uint) UpdateChan

	// Cancel subscription
	Unsubscribe(ch UpdateChan)

	// Return address balance
	GetBalance(address string) int64

	// Get current blockchain height
	GetHeight() (height int64)

	// Return true if manager synced with bitcoind
	Synced() (sync bool)

	// Safely stop block manager
	Stop()
}

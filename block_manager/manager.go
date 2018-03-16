package block_manager

import (
	"github.com/secnot/gobalance/primitives"
)

// Subscriber updates types
type UpdateClass int

const (
	// Balance request channel size
	BalanceRequestQueueSize = 20
)

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

// BalanceRequest is used to send balance requests to the crawler
// throug BalanceChan channel
type BalanceRequest struct {

	// Bitcoin address
	Address string
	
	// Channel used to send the response
	Resp chan BalanceResponse
}

// Balance request response
type BalanceResponse struct {

	// Bitcoin address balance or 0 if not found
	Balance int64

	// Error generated while processing request
	Err error
}

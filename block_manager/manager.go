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

// manager channels
var (
	// New subscription channel
	SubscribeChan   = make(chan UpdateChan, 10)

	// Unsubscribe existing subscription channel
	UnsubscribeChan = make(chan UpdateChan)

	// Signal crawler to start fetching.
	StartChan       = make(chan chan bool)

	// Signal crawler to stop fetching and exit.
	StopChan        = make(chan chan bool)

	// Balance request channel
	BalanceChan     = make(chan BalanceRequest, BalanceRequestQueueSize)

	// Height request channel 
	HeightChan      = make(chan chan int64)

	// Sync status request channel
	SyncChan        = make(chan chan bool)
)


// Subscribe to crawler helper that returns channel where updates are sent
func Subscribe(chanSize uint) UpdateChan {
	ch := make(UpdateChan, int(chanSize))
	SubscribeChan <- ch
	return ch
}

// Unsubscribe from crawler
func Unsubscribe(ch UpdateChan) {
	UnsubscribeChan <- ch
}

// GetBalance sends a request for the balance of an address and returns the channel
// where the response will be sent
func GetBalance(address string) int64 {

	// Channel that will be used by crawler to send the response
	responseCh := make(chan BalanceResponse)
	BalanceChan <- BalanceRequest{ Address: address, Resp: responseCh}
	balance := <- responseCh
	return balance.Balance
}

// GetHeight returs current height
func GetHeight() int64 {
	responseCh := make(chan int64)
	HeightChan <- responseCh
	height := <- responseCh
	return height
}

// Wait until the manager is synced
func Synced() bool {
	responseChan := make(chan bool)
	SyncChan <- responseChan

	return <- responseChan
}

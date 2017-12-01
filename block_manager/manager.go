package block_manager

import (
	"log"
	"errors"

	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/crawler"
	"github.com/secnot/gobalance/block_manager/storage"
)

// Subscriber updates types
type UpdateClass int

const (
	// Balance request channel size
	BalanceRequestQueueSize = 20
)

const (
	OP_NEWBLOCK  UpdateClass = iota
	OP_BACKTRACK
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
	SubscribeChan   = make(chan UpdateChan)

	// Unsubscribe existing subscription channel
	UnsubscribeChan = make(chan UpdateChan)

	// Signal crawler to start fetching.
	StartChan       = make(chan chan bool)

	// Signal crawler to stop fetching and exit.
	StopChan        = make(chan chan bool)

	// Balance request channel
	BalanceChan     = make(chan BalanceRequest, BalanceRequestQueueSize)

	// Sync status request channel
	SyncChan        = make(chan chan bool)
)

// processUpdate handles raw block updates from crawler
func processUpdate(update crawler.BlockUpdate, manager *BlockManager) (BlockUpdate, error){
	
	switch update.Class {
	case crawler.OP_NEWBLOCK:
		block, err := manager.AddBlock(update.Block, update.Hash)
		return  NewBlockUpdate(OP_NEWBLOCK, block), err
	
	case crawler.OP_BACKTRACK:
		block, err := manager.BacktrackBlock()
		return NewBlockUpdate(OP_BACKTRACK, block), err

	default:
		err := errors.New("Unknown BlockUpdate Class")
		return NewBlockUpdate(OP_NEWBLOCK, nil), err
	}
}

// Block Manager routine
func Manager(sto storage.Storage, commitSize int, confirmations uint16, blockUpdateChan crawler.UpdateChan, sync bool) {

	manager, _ := NewBlockManager(sto, commitSize, confirmations, sync)

	// Block updates subscribers 
	var subscribers = make(map[UpdateChan]bool)

	// Start logging routine for new blocks and backtracks
	go Logger()

	// Accept subscriptions and wait until the start signal is received
	// Fetch blocks until the stop signal is received
	for {
		select {		
			// Subscription request
			case subscriber := <-SubscribeChan:
				subscribers[subscriber] = true

			// Unsusbription request
			case subscriber := <-UnsubscribeChan:
				delete(subscribers, subscriber)

			// Stop crawler and exit
			case ch := <-StopChan:
				ch <- true	// signal stopped
				break

			// New block available
			case update := <- blockUpdateChan:
				
				blockUpdate, err := processUpdate(update, manager)
				if err != nil {
					log.Panic(err)
					return
				}

				for subscriber, _ := range subscribers {
					subscriber <- blockUpdate
				}

			// Request balance for one address <- TODO: Move to block manager
			case req := <-BalanceChan:
				balance, err := manager.GetBalance(req.Address)
				req.Resp <- BalanceResponse{Balance: balance, Err: err}

			case ch := <-SyncChan:
				ch <- manager.Synced()
		}
	}
}


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

// Start starts crawler crawling :), 
func Start() {
	ch := make(chan bool)
	StartChan <- ch

	// Wait until it has started
	<- ch
}


// Stop crawler blocks until successfull exit
func Stop() {	
	ch := make(chan bool)
	StopChan <- ch

	// Wait until it has stopped
	<- ch
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

// Wait until the manager is synced
func Synced() bool {
	responseChan := make(chan bool)
	SyncChan <- responseChan

	return <- responseChan
}

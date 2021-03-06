package crawler


import (
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	
	"github.com/secnot/gobalance/primitives/queue"
)


const (
	// Max Buffered blocks
	FetcherBlockBufferSize = 50	
	
	// Max number of blocks 
	BlockQueueSize = 500
)

// Subscriber updates types
type UpdateClass int

const (
	OP_NEWBLOCK  UpdateClass = iota
	OP_BACKTRACK
)

// Struct used to send chain updates to subscribers
type BlockUpdate struct {
	Class  UpdateClass
	Block  *wire.MsgBlock
	Hash   *chainhash.Hash
	Height uint64
}

//
func NewBlockUpdate(class UpdateClass, block *wire.MsgBlock, hash *chainhash.Hash, height uint64) BlockUpdate{
	return BlockUpdate{
		Class:  class,
		Block:  block,
		Hash:   hash,
		Height: height,
	}
}

// Channel type used to send subscriber updates
type UpdateChan chan BlockUpdate


type Crawler struct {
	// Channel to signal fetcher routine to stop
	fetcherStop chan bool

	// Channel for the reception of fetcher blocks
	fetcherBlocks chan blockRecord

	// Block updates subscribers 
	subscribers map[UpdateChan]bool

	// Height for the next block to retrieve
	height uint64

	// Configuration for bitcoind RPC server
	rpcConfig rpcclient.ConnConfig

	// Hashes for the last n unconfirmed blocks
	blockQueue *queue.Queue
	
	// Crawler interface channels
	//////////////////////////////
	// New subscription channel
	subscribeChan   chan UpdateChan

	// Unsubscribe existing subscription channel
	unsubscribeChan chan UpdateChan

	// Signal crawler to start fetching.
	startChan       chan chan bool

	// Signal crawler to stop fetching and exit.
	stopChan        chan chan bool
}

// NewCrawler creates a new crawler
func NewCrawler(config rpcclient.ConnConfig, startHeight uint64, prevBlockHash chainhash.Hash) (*Crawler, error) {

	blockQueue := queue.New()
	blockQueue.PushBack(prevBlockHash)


	craw := &Crawler{
		fetcherStop:   nil,
		fetcherBlocks: nil,
		rpcConfig:     config,
		height:        startHeight,
		subscribers:   make(map [UpdateChan]bool),
		blockQueue:    blockQueue,

		//
		subscribeChan:   make(chan UpdateChan),
		unsubscribeChan: make(chan UpdateChan),
		startChan:       make(chan chan bool),
		stopChan:        make(chan chan bool),
	}

	go craw.crawlerRoutine()

	return craw, nil
}

// backtrackBlock backtracks and discards last block and restart fetcher 
func (c *Crawler) backtrackBlock() chainhash.Hash {	
	// BACKTRACK ONE BLOCK
	if c.blockQueue.Len() == 1 {
		log.Panic("Backtrack limit reached")
	}

	c.height -= 1
	c.newFetcher(c.height) // Fetch previous block again
	return c.blockQueue.PopBack().(chainhash.Hash)
}

// newBlock adds a new block
func (c *Crawler) newBlock(block *wire.MsgBlock, hash *chainhash.Hash) {	
	c.height += 1
	c.blockQueue.PushBack(*hash)

	if c.blockQueue.Len() > BlockQueueSize {
		c.blockQueue.PopFront()
	}
}

// processBlock process new blockchain block
func (c *Crawler) processBlock(block *wire.MsgBlock, blockHash *chainhash.Hash) {

	// Verify the block hash and retry if there was a transmission error
	verifiedHash := block.BlockHash()
	if verifiedHash != *blockHash {
		log.Print("Crawler: Invalid hash ", *blockHash, verifiedHash)
		c.newFetcher(c.height+1) // Fetch same block again
	}

	if block.Header.PrevBlock != c.blockQueue.Back().(chainhash.Hash) {
		backtrackedBlockHash := c.backtrackBlock()

		// Send backtrack update to subscribers
		c.notifySubscribers(NewBlockUpdate(OP_BACKTRACK, nil, &backtrackedBlockHash, c.height))
	} else {
		c.newBlock(block, &verifiedHash)

		// Send new block to subscribers
		c.notifySubscribers(NewBlockUpdate(OP_NEWBLOCK, block, &verifiedHash, c.height-1))
	}
}

// newFetcher stops current fetcher routine and creates a new one starting at a
// given height
func (c *Crawler) newFetcher(height uint64) {
	
	// Stop previous fetcher
	if c.fetcherStop != nil {
		c.fetcherStop <- true
	}
	
	// Both channels to be closed by fetcher task
	c.fetcherStop   = make(chan bool)
	c.fetcherBlocks = make(chan blockRecord, FetcherBlockBufferSize)

	//
	go fetcher(c.rpcConfig, height, c.fetcherBlocks, c.fetcherStop)
}

// notifySubscribers sends a block update to all the subscribers
func (c *Crawler) notifySubscribers(update BlockUpdate) {
	for subscriber, _ := range c.subscribers {
		subscriber <- update
	}
}

// subscribe adds a subscriber to crawler block updates
func (c *Crawler) addSubscriber(subscriber UpdateChan) {
	c.subscribers[subscriber] = true
}

// unsubscribe removes a subscriber from crawler block updates
func (c *Crawler) delSubscriber(subscriber UpdateChan) {
	delete(c.subscribers, subscriber)
}

// Crawler routine
func (c *Crawler) crawlerRoutine() {

	// Accept subscriptions and wait until the start signal is received
	// Fetch blocks until the stop signal is received
	for {
		select {		
			// Subscription request
			case subscriber := <-c.subscribeChan:
				c.addSubscriber(subscriber)

			// Unsusbription request
			case subscriber := <-c.unsubscribeChan:
				c.delSubscriber(subscriber)

			// Start crawler
			case ch := <-c.startChan:
				if c.fetcherBlocks == nil {
					c.newFetcher(c.height) // Fetch next block
				}
				ch <- true // Signal started

			// Stop crawler and exit
			case ch := <-c.stopChan:
				// TODO: Force commit to db, close fetcher, close channels, etc...
				ch <- true	// signal stopped
				break

			// New block available
			case record := <-c.fetcherBlocks:
				c.processBlock(record.Block, record.BlockHash)
		}
	}
}

// Subscribe to crawler helper that returns channel where updates are sent
func (c *Crawler) Subscribe(chanSize uint) UpdateChan {
	ch := make(UpdateChan, int(chanSize))
	c.subscribeChan <- ch
	return ch
}

// Unsubscribe from crawler
func (c *Crawler) Unsubscribe(ch UpdateChan) {
	c.unsubscribeChan <- ch
}

// Start starts crawler crawling :), 
func (c *Crawler) Start() {
	ch := make(chan bool)
	c.startChan <- ch

	// Wait until it has started
	<- ch
}

// Stop crawler blocks until successfull exit
func (c *Crawler) Stop() {	
	ch := make(chan bool)
	c.stopChan <- ch

	// Wait until it has stopped
	<- ch
}

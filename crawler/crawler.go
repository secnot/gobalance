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

// Crawler channels
var (
	// New subscription channel
	SubscribeChan   = make(chan UpdateChan)

	// Unsubscribe existing subscription channel
	UnsubscribeChan = make(chan UpdateChan)

	// Signal crawler to start fetching.
	StartChan       = make(chan chan bool)

	// Signal crawler to stop fetching and exit.
	StopChan        = make(chan chan bool)
)

type CrawlerData struct {
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
}

// NewCrawler creates a new crawler
func newCrawlerData(config rpcclient.ConnConfig, startHeight uint64, prevBlockHash chainhash.Hash) (*CrawlerData, error) {

	blockQueue := queue.New()
	blockQueue.PushBack(prevBlockHash)


	return &CrawlerData{
		fetcherStop:   nil,
		fetcherBlocks: nil,
		rpcConfig:     config,
		height:        startHeight,
		subscribers:   make(map [UpdateChan]bool),
		blockQueue:    blockQueue,
	}, nil
}

// backtrackBlock backtracks and discards last block and restart fetcher 
func (c *CrawlerData) backtrackBlock() chainhash.Hash {	
	// BACKTRACK ONE BLOCK
	if c.blockQueue.Len() == 1 {
		log.Panic("Backtrack limit reached")
	}

	c.height -= 1
	c.newFetcher(c.height) // Fetch previous block again
	return c.blockQueue.PopBack().(chainhash.Hash)
}

// newBlock adds a new block
func (c *CrawlerData) newBlock(block *wire.MsgBlock, hash *chainhash.Hash) {	
	c.height += 1
	c.blockQueue.PushBack(*hash)

	if c.blockQueue.Len() > BlockQueueSize {
		c.blockQueue.PopFront()
	}
}

// processBlock process new blockchain block
func (c *CrawlerData) processBlock(block *wire.MsgBlock, blockHash *chainhash.Hash) {

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
func (c *CrawlerData) newFetcher(height uint64) {
	
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
func (c *CrawlerData) notifySubscribers(update BlockUpdate) {
	for subscriber, _ := range c.subscribers {
		subscriber <- update
	}
}

// subscribe adds a subscriber to crawler block updates
func (c *CrawlerData) addSubscriber(subscriber UpdateChan) {
	c.subscribers[subscriber] = true
}

// unsubscribe removes a subscriber from crawler block updates
func (c *CrawlerData) delSubscriber(subscriber UpdateChan) {
	delete(c.subscribers, subscriber)
}

// Crawler routine
func Crawler(config rpcclient.ConnConfig, startHeight uint64, prevBlockHash chainhash.Hash) {

	crawler, _ := newCrawlerData(config, startHeight, prevBlockHash)

	// Accept subscriptions and wait until the start signal is received
	// Fetch blocks until the stop signal is received
	for {
		select {		
			// Subscription request
			case subscriber := <-SubscribeChan:
				crawler.addSubscriber(subscriber)

			// Unsusbription request
			case subscriber := <-UnsubscribeChan:
				crawler.delSubscriber(subscriber)

			// Start crawler
			case ch := <-StartChan:
				if crawler.fetcherBlocks == nil {
					crawler.newFetcher(crawler.height) // Fetch next block
				}
				ch <- true // Signal started

			// Stop crawler and exit
			case ch := <-StopChan:
				// TODO: Force commit to db, close fetcher, close channels, etc...
				ch <- true	// signal stopped
				break

			// New block available
			case record := <-crawler.fetcherBlocks:
				crawler.processBlock(record.Block, record.BlockHash)
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

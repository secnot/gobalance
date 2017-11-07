package crawler


import (
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/crawler/storage"
)


const (
	// Confirmations required for a block to be eligible to commit to storage
	BlockConfirmations = 30

	// Cache size
	TxOutCacheSize = 500000
	
	// Max Buffered blocks
	FetcherBlockBufferSize = 50	

	// Balance request channel size
	BalanceRequestQueueSize = 20
)

// Subscriber updates types
type UpdateClass int

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

	// Channel for handling balance requests
	BalanceChan     = make(chan BalanceRequest, BalanceRequestQueueSize)
)

type CrawlerData struct {
	// start/stop flag
	start bool

	// Used to send stop signal to fetcher routine
	fetcherStop chan bool

	// Reception of fetched blocks
	fetcherBlocks chan blockRecord

	// Block updates subscribers 
	subscribers map[UpdateChan]bool

	// Height for the next block to retrieve
	height uint64

	// Block handler
	blockManager *BlockManager

	// Configuration for bitcoind RPC server
	rpcConfig rpcclient.ConnConfig

	// Hashes for the last unconfirmed blocks
	blockQueue []chainhash.Hash
}

// NewCrawler: Allocate new crawler
func newCrawlerData(config rpcclient.ConnConfig, store storage.Storage) (*CrawlerData, error) {

	lastStoredHeight, lastStoredHash, err := store.GetLastBlock()
	if err != nil {
		return nil, err
	}

	manager, err := NewBlockManager(store, TxOutCacheSize, BlockConfirmations)
	if err != nil {
		return nil, err
	}

	// Start crawling the next block
	blockQueue := make([]chainhash.Hash, 0, BlockConfirmations)
	blockQueue = append(blockQueue, lastStoredHash)

	return &CrawlerData{
		fetcherStop:   nil,
		fetcherBlocks: nil,
		rpcConfig:     config,
		height:        uint64(lastStoredHeight+1),
		subscribers:   make(map [UpdateChan]bool),
		blockManager:  manager,
		blockQueue:    blockQueue,
	}, nil
}

// backtrack discards last block and fetchs it again
func (c *CrawlerData) backtrackBlock(block *wire.MsgBlock, hash *chainhash.Hash) *primitives.Block{	
	// BACKTRACK ONE BLOCK
	if len(c.blockQueue) == 1 {
		log.Print(block.Header.PrevBlock)
		log.Print(c.blockQueue[len(c.blockQueue)-1])
		log.Panic("Crawler: Backtrack limit reached")
	}
	pBlock, err := c.blockManager.BacktrackBlock()
	if err != nil {
		log.Panic(err)
	}

	c.height -= 1
	c.blockQueue = c.blockQueue[:len(c.blockQueue)-1]
	c.newFetcher(c.height) // Fetch previous block again
	return pBlock
}

// newblock adds a new block
func (c *CrawlerData) newBlock(block *wire.MsgBlock, hash *chainhash.Hash) *primitives.Block {	
	// ADD NEW BLOCK
	pBlock, err := c.blockManager.AddBlock(block, hash)
	if err != nil {
		log.Panic(err)
	}
	c.height += 1
	c.blockQueue = append(c.blockQueue, *hash)
	return pBlock
}


// processBlock process new blockchain block
func (c *CrawlerData) processBlock(block *wire.MsgBlock, blockHash *chainhash.Hash) {

	// Verify the block hash and retry if there was a transmission error
	verifiedHash := block.BlockHash()
	if verifiedHash != *blockHash {
		log.Print("Crawler: Invalid hash ", *blockHash, verifiedHash)
		c.newFetcher(c.height+1) // Fetch same block again
		return
	}

	if block.Header.PrevBlock != c.blockQueue[len(c.blockQueue)-1] {
		pBlock := c.backtrackBlock(block, &verifiedHash)

		// Send backtrack update to subscribers
		c.notifySubscribers(NewBlockUpdate(OP_BACKTRACK, pBlock))
	} else {
		pBlock := c.newBlock(block, &verifiedHash)

		// Send new block to subscribers
		c.notifySubscribers(NewBlockUpdate(OP_NEWBLOCK, pBlock))
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
func Crawler(config rpcclient.ConnConfig, store storage.Storage) {

	crawler, _ := newCrawlerData(config, store)

	// Start logging routine for new blocks and backtracks
	go Logger()

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

			// Request balance for one address
			case req := <-BalanceChan:
				balance, err := crawler.blockManager.GetBalance(req.Address)
				req.Resp <- BalanceResponse{Balance: balance, Err: err}
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

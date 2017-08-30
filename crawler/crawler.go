package crawler


import (
	"log"
	"sync"
	_ "time"

	"github.com/btcsuite/btcrpcclient"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/crawler/storage"
)


const (
	// Max updates waitting to be processed
	UpdateQueueSize = 10

	// Confirmations required for a block to be eligible to commit to storage
	BlockConfirmations = 50

	// Cache size
	TxOutCacheSize = 500000
)

// Subscriber updates types
type UpdateClass int
const (
	OP_NEWBLOCK  UpdateClass = iota
	OP_BACKTRACK
)



type CrawlerObserver interface {
	NewBlock(block *primitives.Block)
	BacktrackBlock(block *primitives.Block)
}

// queued 
type BlockUpdate struct {
	class UpdateClass
	block *primitives.Block
}

func NewBlockUpdate(class UpdateClass, block *primitives.Block) BlockUpdate{
	return BlockUpdate{
		class: class,
		block: block,
	}
}




type Crawler struct {
	sync.Mutex

	// Callback 
	subscribers []CrawlerObserver

	// Height for the next block to retrieve
	height uint64

	// Block handler
	blockManager *BlockManager

	// pending subscriber updates
	updates chan BlockUpdate

	// Configuration for bitcoind RPC server
	rpcConfig btcrpcclient.ConnConfig
}





// NewCrawler: Allocate new crawler
func NewCrawler(config btcrpcclient.ConnConfig, height uint64, store storage.Storage) (*Crawler, error) {
	manager, err := NewBlockManager(store, TxOutCacheSize, BlockConfirmations)
	if err != nil {
		return nil, err
	}

	return &Crawler{
		rpcConfig:    config,
		height:       height,
		subscribers:  make([]CrawlerObserver, 0, 10),
		updates:      make(chan BlockUpdate, UpdateQueueSize),
		blockManager: manager,
	}, nil
}


// TODO: Concurrently retrieve blocks
func (c *Crawler) rpcCrawler() {


	fetch := NewFetcher(c.rpcConfig, c.height)

	for {
		_, block, _, err := fetch.GetNextBlock()

		pBlock, err := c.blockManager.AddBlock(block)
		if err != nil {
			log.Panic(err)
		}
		
		c.updates <- NewBlockUpdate(OP_NEWBLOCK, pBlock)
		
		// TODO: backtrack when needed
	}
}



// Reads updates from updates channel and then send
func (c *Crawler) notifySubscribersRoutine() {
	// TODO: send updates in parellel not sequentially
	for {
		update := <- c.updates
		for _, sub := range c.subscribers {
			if update.class == OP_NEWBLOCK {
				sub.NewBlock(update.block)
			}
			if update.class == OP_BACKTRACK {
				sub.BacktrackBlock(update.block)
			}
		}
	}
}


// Start block crawler and subscriber notification goroutines.
func (c *Crawler) Start() {
	c.Lock()
	c.Unlock()
	go c.rpcCrawler()
	go c.notifySubscribersRoutine()
}

// Stop crawler gracefully
func (c *Crawler) Stop() {
}


// Subscribe to crawler block updates
func (c *Crawler) Subscribe(subs CrawlerObserver) {
	c.Lock()
	c.subscribers = append(c.subscribers, subs)
	c.Unlock()
}




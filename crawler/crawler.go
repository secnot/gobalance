package crawler


import (
	"log"
	"sync"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	rpcConfig rpcclient.ConnConfig

	// Hashes for the last unconfirmed blocks
	blockQueue []chainhash.Hash
}





// NewCrawler: Allocate new crawler
func NewCrawler(config rpcclient.ConnConfig, store storage.Storage) (*Crawler, error) {

	lastHeight, lastHash, err := store.GetLastBlock()
	if err != nil {
		return nil, err
	}

	manager, err := NewBlockManager(store, TxOutCacheSize, BlockConfirmations)
	if err != nil {
		return nil, err
	}

	// Start crawling the next block
	blockQueue := make([]chainhash.Hash, 0, BlockConfirmations)
	blockQueue = append(blockQueue, lastHash)

	return &Crawler{
		rpcConfig:    config,
		height:       uint64(lastHeight+1),
		subscribers:  make([]CrawlerObserver, 0, 10),
		updates:      make(chan BlockUpdate, UpdateQueueSize),
		blockManager: manager,
		blockQueue:   blockQueue,
	}, nil
}


// TODO: Concurrently retrieve blocks
func (c *Crawler) rpcCrawler() {


	fetch := NewFetcher(c.rpcConfig, c.height)

	for {
		blockHash, block, _, err := fetch.GetNextBlock()
		if err != nil {
			log.Print(err)
		}
		
		// Verify the block hash and retry if there was a transmission error
		verifiedHash := block.BlockHash()
		if verifiedHash != *blockHash {
			log.Print("Crawler: Invalid hash ", *blockHash, verifiedHash)
			fetch.setHeight(c.height+1) // Fetch same block again
			continue 
		}

		if block.Header.PrevBlock != c.blockQueue[len(c.blockQueue)-1] {
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

			fetch.setHeight(c.height) // Fetch previous block next
			c.height -= 1
			c.blockQueue = c.blockQueue[:len(c.blockQueue)-1]
			c.updates <- NewBlockUpdate(OP_BACKTRACK, pBlock)
		} else {
			// ADD NEW BLOCK
			pBlock, err := c.blockManager.AddBlock(block, &verifiedHash)
			if err != nil {
				log.Panic(err)
			}
			c.height += 1
			c.blockQueue = append(c.blockQueue, *blockHash)
			c.updates <- NewBlockUpdate(OP_NEWBLOCK, pBlock)
		}
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

func (c *Crawler) Stop() {
}


// Subscribe to crawler block updates
func (c *Crawler) Subscribe(subs CrawlerObserver) {
	c.Lock()
	c.subscribers = append(c.subscribers, subs)
	c.Unlock()
}




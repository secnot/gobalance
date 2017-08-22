package crawler


import (
	"log"
	"sync"
	"time"

	"github.com/phf/go-queue/queue"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcrpcclient"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/secnot/gobalance/primitives"
)


const (
	// Max updates waitting to be processed
	UpdateQueueSize = 10

	// Blocks logged in case backtrack, or put another way max number
	// of backtracks allowed.
	BacktrackLogSize = 100
)

// Types of pending operations
type UpdateClass int


// Subscriber updates types
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

	// Cached TxOuts 
	cache *TxOutCache

	// Callback 
	subscribers []CrawlerObserver

	// Height for the next block to retrieve
	height uint64

	// Last N blocks (type: primitives.Block)
	blockLog *queue.Queue

	// pending subscriber updates
	updates chan BlockUpdate

	// Configuration for bitcoind RPC server
	rpcConfig btcrpcclient.ConnConfig
}





// NewCrawler: Allocate new crawler
func NewCrawler(config btcrpcclient.ConnConfig, height uint64) *Crawler {
	return &Crawler{
		cache:       NewTxOutCache(),
		rpcConfig:   config,
		height:      height,
		subscribers: make([]CrawlerObserver, 0, 10),
		updates:     make(chan BlockUpdate, UpdateQueueSize),
		blockLog:    queue.New(),
	}
}


// TODO: Concurrently retrieve blocks
func (c *Crawler) rpcCrawler() {


	fetch := NewFetcher(c.rpcConfig, c.height)

	for {
		blockHash, block, height, err := fetch.GetNextBlock()

		// Generate primitive.Block using txoutcache
		pBlock, err := c.processBlock(blockHash, block, height)
		if err != nil {
			log.Panic(err)
		}
		
		// Queue new update and remove oldest block from log
		c.Lock()
		if c.blockLog.Len() > BacktrackLogSize {
			c.blockLog.PopFront()
		}
		c.blockLog.PushBack(pBlock)
		c.height++
		c.updates <- NewBlockUpdate(OP_NEWBLOCK, pBlock)
		c.Unlock()
	}
}


// processTx
func (c *Crawler) processTx(wireTx *wire.MsgTx) (*primitives.Tx, error){	

	hash := wireTx.TxHash()
	c.cache.SetTx(&hash, wireTx)
	tx := primitives.NewTx(&hash)
	
	// Outputs
	for n, _ := range wireTx.TxOut {
		txout := c.cache.PeekTxOut(&hash, uint32(n))
		tx.AddOut(txout)	
	}

	// If it is coinbase transaction it has only one input
	if blockchain.IsCoinBaseTx(wireTx) {
		return tx, nil
	}

	// Inputs
	for _, txin := range wireTx.TxIn {
		// If the  txin is 
		txout := c.cache.GetTxOut(&txin.PreviousOutPoint.Hash, txin.PreviousOutPoint.Index)
		if txout != nil {
			tx.AddIn(txout)
		}
	}

	return tx, nil
}

// processBlock generates a primitive.Block from wire.MsgBlock
func (c *Crawler) processBlock(blockHash *chainhash.Hash, block *wire.MsgBlock, height uint64) (*primitives.Block, error) {
	prevHash := block.Header.PrevBlock

	// Verify block hash
	verifiedHash := block.BlockHash()
	if verifiedHash != *blockHash {
		log.Panic(verifiedHash, blockHash)
	}


	transactions := make([]*primitives.Tx, len(block.Transactions))

	for n, wireTx := range block.Transactions {
		tx, err := c.processTx(wireTx)
		if err != nil {
			return nil, err
		}
		transactions[n] = tx
	}
	pBlock := primitives.NewBlock(*blockHash, prevHash, height)
	pBlock.Transactions = transactions

	if pBlock.Height == 200000 {
		time.Sleep(100000*time.Millisecond)
	}
	return pBlock, nil
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




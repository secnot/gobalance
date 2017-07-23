package crawler


import (
	"log"
	"sync"
	"time"

	"github.com/phf/go-queue/queue"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcrpcclient"
	"github.com/secnot/gobalance/primitives"
)


const (
	// Delay between failed request (in milliseconds)
	RPCRetryDelay = 4000

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

	// Initialized rpc client 
	client *btcrpcclient.Client

	// Callback 
	subscribers []CrawlerObserver

	// Height for the next block to retrieve
	height uint64

	// Top block height (last update)
	top uint64

	// Last N blocks (type: primitives.Block)
	blockLog *queue.Queue

	// pending subscriber updates
	updates chan BlockUpdate
}





// NewCrawler: Allocate new crawler
func NewCrawler(client *btcrpcclient.Client, height uint64) *Crawler {
	return &Crawler{
		cache:       NewTxOutCache(client),
		client:      client,
		height:      height,
		top:         0, 
		subscribers: make([]CrawlerObserver, 0, 10),
		updates:     make(chan BlockUpdate, UpdateQueueSize),
		blockLog:    queue.New(),
	}
}


// TODO: Concurrently retrieve blocks
func (c *Crawler) rpcCrawler() {

	retries := 0
	for {
		if retries > 0 {
			time.Sleep(RPCRetryDelay*time.Millisecond)
		}

		// If the height for the next block has passed the current top, poll
		// for the new top until there is a new block
		c.Lock()
		top, height := c.top, c.height
		c.Unlock()
		if top < height {
			if !c.updateBlockchainHeight() {
				retries++
				continue
			}
		}

		// Fetch the next block
		block, err := c.getBlock(height)
		if err != nil {
			log.Print(err)
			retries++
			continue
		}

		retries = 0 // Not a connection failure from this point on

		// Check the new block is part of the current chain, if not
		// backtrack one block and try again.
		c.Lock()
		lastBlock := c.blockLog.Back()
		if lastBlock != nil && block.Header.PrevBlock != lastBlock.(*primitives.Block).Hash {
			c.updates <- NewBlockUpdate(OP_BACKTRACK, lastBlock.(*primitives.Block))
			c.blockLog.PopBack()
			c.height--
			c.Unlock()
			continue
		}
		c.Unlock()

		// Generate primitive.Block using txoutcache
		pBlock, err := c.processBlock(block, height)
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


// update crawler blockchain top block height, returns true if the top was updated
func (c *Crawler) updateBlockchainHeight() (updated bool) {
	blockCount, err := c.client.GetBlockCount()
	if err != nil {
		log.Print(err)
		return false
	}

	c.Lock()
	if uint64(blockCount) > c.top {
		c.top = uint64(blockCount)
		updated = true
	} else {	
		updated = false
	}
	c.Unlock()
	return
}

// get block of given height
func (c *Crawler) getBlock(height uint64) (*wire.MsgBlock, error) {		
	
	blockHash, err := c.client.GetBlockHash(int64(c.height))
	if err != nil {
		return nil, err
	}

	block, err := c.client.GetBlock(blockHash)
	if err != nil {
		return nil, err
	}
	
	return block, nil
}

// processTx
func (c *Crawler) processTx(wireTx *wire.MsgTx) (*primitives.Tx, error){	

	hash := wireTx.TxHash()
	c.cache.SetTx(&hash, wireTx)
	tx := primitives.NewTx(&hash)
	
	// Outputs
	for n, _ := range wireTx.TxOut {
		txout, err := c.cache.PeekTxOut(&hash, uint32(n))
		if err != nil {
			return nil, err
		}
		tx.AddOut(txout)	
	}

	// If it is coinbase transaction it has only one input
	if blockchain.IsCoinBaseTx(wireTx) {
		return tx, nil
	}

	// Inputs
	for _, txin := range wireTx.TxIn {
		// If the  txin is 
		txout, err := c.cache.GetTxOut(&txin.PreviousOutPoint.Hash, txin.PreviousOutPoint.Index)
		if err != nil {
			return nil, err
		}
	
		tx.AddIn(txout)
	}

	return tx, nil
}

// processBlock generates a primitive.Block from wire.MsgBlock
func (c *Crawler) processBlock(block *wire.MsgBlock, height uint64) (*primitives.Block, error) {
	prevHash := block.Header.PrevBlock
	hash := block.BlockHash()

	transactions := make([]*primitives.Tx, len(block.Transactions))

	for n, wireTx := range block.Transactions {
		tx, err := c.processTx(wireTx)
		if err != nil {
			return nil, err
		}
		transactions[n] = tx
	}

	pBlock := primitives.NewBlock(hash, prevHash, height)
	pBlock.Transactions = transactions

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


// Start crawling goroutine at given height
func (c *Crawler) Start() {
	c.Lock()
	c.Unlock()
	go c.rpcCrawler()
	go c.notifySubscribersRoutine()
}

func (c *Crawler) Stop() {
}


// Attach another subscriber to crawler block updates
func (c *Crawler) Subscribe(subs CrawlerObserver) {
	c.Lock()
	c.subscribers = append(c.subscribers, subs)
	c.Unlock()
}




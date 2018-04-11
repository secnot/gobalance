package block_manager


import (
	"log"
	"time"
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/secnot/gobalance/primitives"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/secnot/gobalance/crawler"
	"github.com/secnot/gobalance/interfaces"
	"github.com/secnot/gobalance/block_manager/storage"
)

const (
	// Average number of transaction inputs in a block
	AverageBlockInputs = 2048 * 10 // Transactions * Inputs

	// Block commit delay
	DefaultCommitDelay = 10*time.Second

	//
	BalanceRequestQueueSize = 100

	// Size of channel for crawler updates
	CrawlerUpdateChanSize = 50
)

var ErrBacktrackLimit = errors.New("Backtrack limit reached")


// BalanceRequest is used to send balance requests to the manager
// through BalanceChan
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


type BlockManager struct {

	// Sync mode flag, when in sync mode the balance is disabled to save memory
	// and for a faster syncing
	Sync bool
	
	// Max number of txout cached in memory before a commit is required
	CommitSize int

	// Min number of blocks before a commit is recommended
	CommitMinBlocks int

	// Added Delay between when a commit is ready and it's start, the purpose together
	// with CommitMinBlocks is to assure not may nodes start a commit at the same time
	// (this delay is ignore when in sync mode)
	CommitDelay time.Duration

	// Confirmations required for a block to be elegible for storage
	Confirmations uint16

	// Last block height
	height int64

	// Last time a block was added 
	lastTime time.Time

	// Last 
	topHeight uint64

	// 
	storageCache *storage.StorageCache

	// 
	crawler *crawler.Crawler

	// Channel for receiving crawler updates
	crawlerUpdateChan crawler.UpdateChan

	// Blocks waiting for enough confirmations before committing to storage
	pendingBlocks *primitives.BlockQueue

	//
	commitTimer *time.Timer
	commitTimerStartedFlag bool
	
	// subscriberts
	subscribers map[interfaces.UpdateChan] bool
	
	// CONTROL CHANNELS

	// New subscription channel
	SubscribeChan   chan interfaces.UpdateChan

	// Unsubscribe existing subscription channel
	UnsubscribeChan chan interfaces.UpdateChan

	// Signal crawler to start fetching.
	StartChan       chan chan bool

	// Signal crawler to stop fetching and exit.
	StopChan        chan chan bool

	// Balance request channel
	BalanceChan     chan BalanceRequest

	// Height request channel 
	HeightChan      chan chan int64

	// Sync status request channel
	SyncChan        chan chan bool
}

// Start initializes and launches BlockManager routines
func (b *BlockManager) Start(sto storage.Storage, crawlerM *crawler.Crawler) error {

	cache, err := storage.NewStorageCache(sto, !b.Sync)
	if err != nil {
		return err
	}

	b.crawler = crawlerM
	b.crawlerUpdateChan = crawlerM.Subscribe(CrawlerUpdateChanSize)

	if b.CommitMinBlocks < 1 {
		b.CommitMinBlocks = 1
	}

	if b.CommitSize < 1 {
		b.CommitSize = 1
	}

	b.storageCache = cache
	b.height       = cache.GetHeight()
	
	// Initialize subscribers
	b.subscribers = make(map[interfaces.UpdateChan]bool)

	// Initialize channels
	b.SubscribeChan   = make(chan interfaces.UpdateChan, 10)
	b.UnsubscribeChan = make(chan interfaces.UpdateChan)
	b.StartChan       = make(chan chan bool)
	b.StopChan        = make(chan chan bool)
	b.BalanceChan     = make(chan BalanceRequest, BalanceRequestQueueSize)
	b.HeightChan      = make(chan chan int64)
	b.SyncChan        = make(chan chan bool)
	
	// Initialize timer so its channel can be added to select loop, but stop signal
	b.commitTimer  = time.NewTimer(10*time.Second)
	b.commitTimer.Stop()
	b.commitTimerStartedFlag = false
	
	// Queue
	b.pendingBlocks = primitives.NewBlockQueue()

	// Launch main routine
	go b.managerRoutine()
	
	return nil
}

// getBlockIntpus populates transactions inputs address and value
func (b *BlockManager) getBlockInputs(block *primitives.Block) (*primitives.Block, error) {	
	
	// map all the block transactions
	blockTxIdx := make(map[chainhash.Hash]*primitives.Tx, len(block.Transactions))
	for _, tx := range block.Transactions {
		blockTxIdx[*tx.Hash] = tx
	}
	
	// Find all the missing transaction inputs to retrieve them in a single bulk 
	// operartion
	missingIns := make([]storage.TxOutId, 0, AverageBlockInputs)
	
	for _, tx := range block.Transactions {

		if tx.IsCoinBase() {
			continue
		}

		// Add to missing inputs all the ones not queued or from the current block
		for _, txIn := range tx.In {

			txInBlock := blockTxIdx[*txIn.TxHash]
			txInQueue, _ := b.pendingBlocks.Tx(*txIn.TxHash)

			if txInBlock == nil && txInQueue == nil {
				missingIns = append(missingIns, storage.TxOutId{
					TxHash: *txIn.TxHash, Nout: txIn.Nout,})
			}
		}
	}

	// Get missing transaction outputs from storage
	missingData, err := b.storageCache.BulkGetTxOut(missingIns)
	if err != nil {
		return nil, err
	}

	// Add data to transactions inputs
	var missingIdx int = 0 // Index for the next unused missing input

	for _, tx := range block.Transactions {
	
		if tx.IsCoinBase() {
			continue
		}

		for _, txIn := range tx.In {
			if tx, _ := b.pendingBlocks.Tx(*txIn.TxHash); tx != nil {
				// The input is an output from a pending block
				txIn.Addr  = tx.Out[txIn.Nout].Addr
				txIn.Value = tx.Out[txIn.Nout].Value
			} else if tx := blockTxIdx[*txIn.TxHash]; tx != nil {
				// The input is an output from the current block
				txIn.Addr  = tx.Out[txIn.Nout].Addr
				txIn.Value = tx.Out[txIn.Nout].Value
			} else {
				// The input was retrieved from storage
				txIn.Addr  = missingData[missingIdx].Addr
				txIn.Value = missingData[missingIdx].Value
				missingIdx += 1
			}
		}
	}	

	return block, nil
}

// buildTx returns a primitives.Tx for the MsgTx (without inputs addresses and values)
func (b *BlockManager) buildTx(wireTx *wire.MsgTx) *primitives.Tx {
	hash := wireTx.TxHash()
	tx := primitives.NewTx(&hash)

	for nout, txOut := range wireTx.TxOut {
		address := primitives.PkScriptToAddr(txOut.PkScript)
		txout := primitives.NewTxOut(&hash, uint32(nout), address, txOut.Value)
		if txout != nil {
			tx.AddOut(txout)
		}
	}

	for _, txIn := range wireTx.TxIn {
		prevOutHash := txIn.PreviousOutPoint.Hash
		txin := primitives.NewTxOut(&prevOutHash, uint32(txIn.PreviousOutPoint.Index), "", 0)
		if txin != nil {
			tx.AddIn(txin)
		}
	}

	return tx
}

func (b *BlockManager) buildBlock(bHash *chainhash.Hash, block *wire.MsgBlock, height uint64) (*primitives.Block, error) {
	
	// Build all the block transactions (without inputs)
	transactions := make([]*primitives.Tx, 0, len(block.Transactions))
	for _, wireTx := range block.Transactions {
		tx := b.buildTx(wireTx)
		transactions = append(transactions, tx)
	}

	pBlock := primitives.NewBlock(*bHash, block.Header.PrevBlock, height)
	pBlock.Transactions = transactions

	return pBlock, nil
}

// TimeSinceLastBlock returns the seconds elapsed since the last block was processed
func (b *BlockManager) timeSinceLastBlock() float64 {
	now := time.Now()
	return now.Sub(b.lastTime).Seconds() // time since last block
}

// AddBlock adds a wire.Block to the manager returning primitives.Block equivalent
func (b *BlockManager) addBlock(block *wire.MsgBlock, blockHash *chainhash.Hash) (*primitives.Block, error) {
	
	// Generate	block and add it to pending of confirmation block queue
	pBlock, err := b.buildBlock(blockHash, block, uint64(b.height+1))
	if err != nil {
		return nil, err
	}

	if !b.Sync {
		pBlock, err = b.getBlockInputs(pBlock)
		if err != nil {
			return nil, err
		}
	}

	b.pendingBlocks.PushBack(pBlock)

	// Update current height
	b.height += 1

	// Add confirmed block and height to storage cache
	if b.pendingBlocks.Len() > int(b.Confirmations) {
		confirmedBlock := b.pendingBlocks.PopFront()
		b.storageCache.AddBlock(confirmedBlock)
	}

	return pBlock, nil
}

// BacktrackBlock backtracks and returns last block
func (b *BlockManager) backtrackBlock() (*primitives.Block, error){
	if b.pendingBlocks.Len() == 0 {
		return nil, ErrBacktrackLimit
	}

	b.height -= 1
	block := b.pendingBlocks.PopBack()
	return block, nil
}

// UncommittedBlocks returns the number of blocks confirmed and already in storage
// cache but not yet committed
func  (b *BlockManager) uncommittedBlocks() int {
	return b.storageCache.UncommittedBlocks()
}


// topReached returns true if the last block is the top of the blockchain
func (b *BlockManager) topReached() bool {

	// Check against the stored top
	if uint64(b.height) < b.topHeight {
		return false
	}

	// Check against the current blockchain top
	b.topHeight = b.crawler.TopHeight()
	return uint64(b.height) >= b.topHeight
}

// CommitRequired returns true if it's time for a commit
func (b *BlockManager)commitRequired() bool {
	
	// Commit already scheduled
	if b.commitTimerStartedFlag {
		return false
	}

	// Check there is something to commit
	if b.uncommittedBlocks() < 1 {
		return false
	}

	// If the max cache size has been reached is time to commit
	if b.storageCache.UncommittedLen() > b.CommitSize {
		return true
	}

	// Sync mode: we need to commit the last blocks as soon as the top of the 
	// chain is reached.
	if b.Sync {
		return b.topReached()
	}

	// Normal mode: commit when min uncommited blocks are reached
	if b.uncommittedBlocks() > b.CommitMinBlocks {
		return true
	}

	return false
}

// Synced returns true when the manager is with the last blockchain block
func (b *BlockManager) synced() bool {
	
	// Check all is committed
	if b.commitTimerStartedFlag {
		return false
	}

	if b.storageCache.UncommittedBlocks() > 0 {
		return false
	}

	return b.topReached()
}

// Commit all cached blocks to storage
func (b *BlockManager) commit() error {	
	log.Print("Commit: ", b.storageCache.GetHeight())
	err := b.storageCache.Commit()
	if err != nil {
		return err
	}
	return nil
}

// GetBalance returns address balance
func (b *BlockManager) getBalance(address string) (int64, error) {
	pendingBalance, ok := b.pendingBlocks.GetBalance(address)
	if !ok {
		pendingBalance = 0
	}
	storedBalance, err  := b.storageCache.GetBalance(address)
	if err != nil  {
		return 0, err
	}

	return pendingBalance + storedBalance, nil
}

// processBlockUpdate handles raw block updates from crawler
func (b *BlockManager) processBlockUpdate(update crawler.BlockUpdate) (interfaces.BlockUpdate, error){
	
	switch update.Class {
	case crawler.OP_NEWBLOCK:
		block, err := b.addBlock(update.Block, update.Hash)
		return  interfaces.NewBlockUpdate(interfaces.OP_NEWBLOCK, block), err
	
	case crawler.OP_BACKTRACK:
		block, err := b.backtrackBlock()
		return interfaces.NewBlockUpdate(interfaces.OP_BACKTRACK, block), err

	default:
		err := errors.New("Unknown BlockUpdate Class")
		return interfaces.NewBlockUpdate(interfaces.OP_NEWBLOCK, nil), err
	}
}

// signalSubscribers
func (b *BlockManager) signalSubscribers(update interfaces.BlockUpdate) {

	for subscriber, _ := range b.subscribers {
		subscriber <- update
	}
}

// startCommitTimer
func (b *BlockManager) startCommitTimer() {

	// Start timer
	if b.Sync { // In Sync mode commit as fast as possible
		b.commitTimer.Reset(10*time.Millisecond)
	} else {
		b.commitTimer.Reset(DefaultCommitDelay+b.CommitDelay)
	}
}

// stopCommitTimer stop commit timer prematurely, WARNING: if it is called after
// the signal channel is read by the main loop it will deadlock
func (b *BlockManager) stopCommitTimer() {
	
	if !b.commitTimer.Stop() {
		<- b.commitTimer.C
	}
}

// Block Manager routine handling block update and other requests
func (b *BlockManager) managerRoutine() {

	// Start logging routine for new blocks and backtracks
	go Logger(b)

	// Accept subscriptions and wait until the start signal is received
	// Fetch blocks until the stop signal is received
	for {
		select {		
			// Subscription request
			case subscriber := <-b.SubscribeChan:
				b.subscribers[subscriber] = true

			// Unsusbription request
			case subscriber := <-b.UnsubscribeChan:
				delete(b.subscribers, subscriber)

			// Current height
			case ch := <- b.HeightChan:
				ch <- b.height

			// Stop crawler and exit
			case ch := <- b.StopChan:
				ch <- true	// signal stopped
				// TODO: Stop logger
				break

			// New block available
			case update := <- b.crawlerUpdateChan:
				blockUpdate, err := b.processBlockUpdate(update)
				if err != nil {
					log.Panic(err)
					return
				}

				b.signalSubscribers(blockUpdate)

				// Need to start a commit?
				if !b.commitTimerStartedFlag && b.commitRequired() {
					// Start commit timer
					b.commitTimerStartedFlag = true
					b.startCommitTimer()

					// Signal subscribers a commit is scheduled
					b.signalSubscribers(interfaces.NewBlockUpdate(interfaces.OP_COMMIT, nil))
				}

				// Record last block arrival time
				b.lastTime = time.Now()

			// Commit timer expired
			case <- b.commitTimer.C:
				b.commitTimerStartedFlag = false
				b.commit()
				b.signalSubscribers(interfaces.NewBlockUpdate(interfaces.OP_COMMIT_DONE, nil))
				b.lastTime = time.Now()

			// Request balance for one address.
			case req := <- b.BalanceChan:
				balance, err := b.getBalance(req.Address)
				req.Resp <- BalanceResponse{Balance: balance, Err: err}

			// Request 
			case ch := <- b.SyncChan:
				ch <- b.synced()
		}
	}
}


// Subscribe to crawler helper that returns channel where updates are sent
func (b *BlockManager) Subscribe(chanSize uint) interfaces.UpdateChan {
	ch := make(interfaces.UpdateChan, int(chanSize))
	b.SubscribeChan <- ch
	return ch
}

// Unsubscribe from crawler
func (b *BlockManager) Unsubscribe(ch interfaces.UpdateChan) {
	b.UnsubscribeChan <- ch
}

// GetBalance sends a request for the balance of an address and returns the channel
// where the response will be sent
func (b *BlockManager) GetBalance(address string) int64 {

	// Channel that will be used by crawler to send the response
	responseCh := make(chan BalanceResponse)
	b.BalanceChan <- BalanceRequest{ Address: address, Resp: responseCh}
	balance := <- responseCh
	close(responseCh)
	return balance.Balance
}

// GetHeight returs current height
func (b *BlockManager) GetHeight() (height int64) {
	responseCh := make(chan int64)
	b.HeightChan <- responseCh
	height = <- responseCh
	close(responseCh)
	return
}

// Wait until the manager is synced
func (b *BlockManager) Synced() (sync bool) {
	responseCh := make(chan bool)
	b.SyncChan <- responseCh
	sync = <-responseCh
	close(responseCh)
	return
}

// Stop crawler blocks until successfull exit
func (b *BlockManager) Stop() {
	doneCh := make(chan bool)
	b.StopChan <- doneCh

	// Wait until it has stopped
	<- doneCh
	close(doneCh)
}



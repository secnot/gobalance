package crawler

import (
	"fmt"
	"log"
	"time"
	"sync"
	"github.com/secnot/gobalance/primitives/queue"
	"github.com/secnot/gobalance/utils"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

const (
	// Delay between failed requests retries (in milliseconds)
	RPCRetryDelay = 4000

	// RPC reconnect delay (in milliseconds)
	RPCReconnectDelay = 30000

	// Max Buffered blocks
	FetcherBlockBufferSize = 50	
)

type blockRecord struct {
	
	// Verified block hash
	BlockHash *chainhash.Hash
	
	// 
	Block *wire.MsgBlock
	
	// Block height for the block at retrieval time.
	Height uint64
}


type Fetcher struct {
	// This mutex only purpose is to lock while updating topHeight 
	sync.RWMutex

	// Height for the top of the blockchain
	topHeight uint64

	// height for the next block to fetch
	height uint64

	// bitcoind server config used to create the clients
	config []rpcclient.ConnConfig

	// Initialized bitcoind clients (queue contains *rpcclient.Client)
	// the first one in the list is the one in use
	clients *queue.Queue

	// Client being used for requests. 
	currentClient *rpcclient.Client

	// Channel where the channels
	UpdatesChan chan blockRecord

	// Channel for stop signals
	stopChan chan chan bool
}


// NewFetcher
func NewFetcher(servers []rpcclient.ConnConfig, height uint64) (*Fetcher, error) {

	// copy original server config
	config := make([]rpcclient.ConnConfig, len(servers))
	copy(config, servers)

	// Randomize order so each peer uses a different initial bitcoind server
	utils.Shuffle(config, utils.NewRandSource())
	
	// Build fetcher
	fetcher := &Fetcher{
		topHeight: height,
		height: height,
		config: config,
		clients: queue.New(),
		stopChan: make(chan chan bool),
		UpdatesChan: make(chan blockRecord),
	}


	// Create clients for all the servers
	for _, srvConfig := range config {
		// The notification parameter is nil since notifications are
		// not supported in HTTP POST mode.
		client, err := rpcclient.New(&srvConfig, nil)
		if err != nil {
			log.Printf("Fetching error: %v\n", err)
		} else {
			fetcher.clients.PushBack(client)
		}
	}

	if fetcher.clients.Len() < 1 {
		return nil, fmt.Errorf("No Bitcoind server available")
	}

	// Initialize currentClient
	fetcher.nextClient()

	// launch routine
	go fetcher.fetcherRoutine()

	// done
	return fetcher, nil
}

// Stop fetcher and wait for confirmation message
func (f *Fetcher) Stop() {
	confirmationCh := make(chan bool)
	f.stopChan <- confirmationCh
	<-confirmationCh
}

// TopHeight returns the height of the top of the blockchain announced by bitcoind server,
// if there is a discrepancy between bitcoind servers the highest one is used
func (f *Fetcher) TopHeight() uint64 {
	f.RLock()
	defer f.RUnlock()

	return f.topHeight
}

// nextClient changes the current client for the next in the list
func (f *Fetcher) nextClient() {
	// Move the client in use to the end of the queue
	current := f.clients.PopFront()
	f.clients.PushBack(current.(*rpcclient.Client))

	// Point currentClient to the new one
	f.currentClient = f.clients.Front().(*rpcclient.Client)
}

// CleanUp before exitting
func (f *Fetcher) cleanUp(confirmationCh chan bool) {
	close(f.stopChan)
	close(f.UpdatesChan)
	
	// This should be required onlu when using btcd
	iter := f.clients.Iter()
	for client, ok := iter.Next(); ok; client, ok = iter.Next() {
		client.(*rpcclient.Client).Shutdown()
	}

	// signal closed
	confirmationCh <- true
}

// Try to retrieve next block from 
func (f *Fetcher) getNextBlock() (record blockRecord, ok bool) {

		// If the top of the blockchain has been reached check if there
		// are new blocks.
		if f.topHeight < f.height {
			blockCount, err := f.currentClient.GetBlockCount()
			if err != nil || uint64(blockCount) <= f.topHeight {
				f.nextClient()
				return blockRecord{}, false
			}
			
			// Lock because it can be accessed directly
			f.Lock()
			f.topHeight = uint64(blockCount)
			f.Unlock()
		}

		// Read next block	
		blockHash, err := f.currentClient.GetBlockHash(int64(f.height))
		if err != nil {
			log.Print("Fetcher: GetBlockHash: ", err)
			f.nextClient()
			return blockRecord{}, false
		}

		block, err := f.currentClient.GetBlock(blockHash)
		if err != nil {
			log.Print("Fetcher: GetBlock ", err)
			f.nextClient()
			return blockRecord{}, false
		}
	
		// Block ready
		f.height += 1
		record = blockRecord{
			BlockHash: blockHash, 
			Height: f.height, 
			Block: block,
		}
		return record, true
}

// fetcherRoutine handles bitcoind requests
func (f *Fetcher) fetcherRoutine() {
	// Main fetching loop
	retry := false        // failed requests retries

	for {

		// If the previous block request wasn't successful, wait retry delay  
		if retry {
			retryTimer := time.NewTimer(RPCRetryDelay*time.Millisecond)
			select {

			// retryTimer expired try again
			case <-retryTimer.C:
				break

			// stop signal received exit
			case confirmationChan := <-f.stopChan:
				retryTimer.Stop()
				f.cleanUp(confirmationChan)
				return
			}

			retry = false
		}

		// Get the block
		record, ok := f.getNextBlock()
		if !ok {
			retry = true
			continue
		}

		// Send block 
		select {
		case confirmationChan := <- f.stopChan:
			f.cleanUp(confirmationChan)
			return
		
		case f.UpdatesChan <- record:
			// Ready for next block
			retry = false
		}
	}
}



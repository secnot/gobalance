package crawler

import (
	"log"
	"time"
	"sync"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcrpcclient"
)


const (
	// Max Buffered blocks
	BlockBufferSize = 10
	
	// Delay between failed requests retries (in milliseconds)
	RPCRetryDelay = 4000

	// RPC reconnect delay (in milliseconds)
	RPCReconnectDelay = 30000
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

	sync.Mutex

	// Used to signal fetching routine to stop (by closing)
	stopSignal chan struct{}

	// Buffered channel for fetched blocks
	blockBuffer <-chan blockRecord

	// Height for the last block returned
	height uint64

	// RPC service cofig
	rpcConfig btcrpcclient.ConnConfig
}


//
func fetchingRoutine(config btcrpcclient.ConnConfig, height uint64, buffer chan<- blockRecord, stop chan struct{}) {

	var client *btcrpcclient.Client
	var err error
	var topHeight uint64 = 0 // Height for the last block in the chain
	
	if height < 0 {
		log.Panic("Requested block with negative height")
	}

	// TODO: Check config in post mode

	// Connect to rpc service
	for {
		// The notification parameter is nil since notifications are
		// not supported in HTTP POST mode.
		client, err = btcrpcclient.New(&config, nil)
		if err == nil {
			break
		}
		
		log.Print(err)
		time.Sleep(RPCReconnectDelay*time.Millisecond)
	}

	// Main fetching loop
	retries := 0	// failed requests retries
	for {
		if retries > 0 {
			time.Sleep(RPCRetryDelay*time.Millisecond)
		}

		// If the top of the blockchain has been reached wait until there
		// is a new block available.
		if topHeight < height {
			blockCount, err := client.GetBlockCount()
			if err != nil || uint64(blockCount) <= topHeight {
				retries++
				continue // Wait and retry
			}
			topHeight = uint64(blockCount)
			retries = 0
		}

		// Read next block	
		blockHash, err := client.GetBlockHash(int64(height))
		if err != nil {
			retries++
			continue // Wait and retry
		}

		block, err := client.GetBlock(blockHash)
		if err != nil {
			retries++
			continue // Wait and retry
		}
	
		// Verify block hash
		// TODO: Use worker pool to verify block hash
		/*
		verifiedHash := block.BlockHash()
		if verifiedHash != *blockHash {
			log.Printf("Block hash validation error %v", blockHash)
			retries++
			continue // Wait and retry
		}
		*/

		// Add the block to the buffer while waitting for a stop signal
		record := blockRecord{
			BlockHash: blockHash, 
			Height: height, 
			Block: block,
		}

		select {
		case <- stop:
			// Close and exit
			close(buffer)
			close(stop)
			client.Shutdown()
			return

		case buffer <- record:
			// Ready for next block
			retries = 0
			height += 1
			log.Print(len(buffer))
		}
	}
}



// Create new fetcher
func NewFetcher(config btcrpcclient.ConnConfig, height uint64) (f *Fetcher) {
	f = &Fetcher {rpcConfig: config, height: height}
	f.setHeight(height)
	return f
}


// Stop current fetchingRoutine and start a new one at another height
func (f *Fetcher) setHeight(height uint64) {
	
	// Channels for the new fetching routine
	signal := make(chan struct{})
	buffer := make(chan blockRecord, BlockBufferSize)
	
	
	f.Lock()

	// Stop old fetching routine
	if f.stopSignal != nil {
		f.stopSignal <- struct{}{}
	}

	f.stopSignal  = signal
	f.blockBuffer = buffer
	f.height = height
	go fetchingRoutine(f.rpcConfig, height, buffer, signal)
	f.Unlock()
}

// Restart fetching at new height by discarding the current buffer and 
// starting a new fetching goroutine. While the current fetching goroutine 
// is closed
func (f *Fetcher) ResetHeight(height uint64) {
	f.setHeight(height)
}

// Retrieve next block in the chain or block until available
func (f *Fetcher) GetNextBlock() (blockHash *chainhash.Hash, block *wire.MsgBlock, height uint64, err error) {		
	record, ok := <- f.blockBuffer 
	if !ok {
		// channel closed and drained
		return f.GetNextBlock()
	}

	return record.BlockHash, record.Block, record.Height, nil
}


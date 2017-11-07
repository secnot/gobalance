package crawler

import (
	"log"
	"time"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

const (
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

//
func fetcher(config rpcclient.ConnConfig, height uint64, buffer chan blockRecord, stop chan bool) {

	var client *rpcclient.Client
	var err error
	var topHeight uint64 = 0 // Height for the last block in the chain
	
	// Connect to rpc service
	for {
		// The notification parameter is nil since notifications are
		// not supported in HTTP POST mode.
		client, err = rpcclient.New(&config, nil)
		if err == nil {
			break
		}
		
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
			log.Print("Fetcher: GetBlock ", err)
			retries++
			continue // Wait and retry
		}
	
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
		}
	}
}



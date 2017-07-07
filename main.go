// Copyright (c) 2014-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"fmt"
	_ "github.com/secnot/simplelru"
	"github.com/btcsuite/btcrpcclient"
	"./balance"
	"time"
	//"github.com/btcsuite/btcd/txscript"
)



//build_block(block *wire.MsgBlock)

/*
type BlockPrefetchCache struct {

	cache []*wire.MsgBlock
	size int
	
	lock
}

*/
/*
func NewBlockPrefetchCache(conConfig btcrpcclient.ConnConfig, size int) *BlockPrefetchCache{
	return
}

func (* BlockPrefetchCache) get_next_block() (*wire.MsgBlock, error){
}
*/



func main() {
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	connCfg := &btcrpcclient.ConnConfig{
		Host:         "localhost:8332",
		User:         "secnot",
		Pass:         "12345",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	// Get the current block count.
	blockCount, err := client.GetBlockCount()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Block count: %d", blockCount)


	// Initialize txpool
	pool := balance.NewTxOutPool()	

	// Get blocks from start to finish
	for i:= int64(0); i<blockCount; i+=1 {
		blockHash, err := client.GetBlockHash(i)
		if err != nil {
			log.Fatal(err)
			return
		}
		block, err := client.GetBlock(blockHash)
		if err != nil {
			log.Fatal(err)
			return
		}
		// block -> *wire.MsgBlock
		//fmt.Printf("Block %d: %v\n", i, block.Hash, block)
		fmt.Printf("Block %d: %v\n", i, block.BlockHash())
		pool.AddBlock(block)
	}
	time.Sleep(10000*time.Second)
}

//class, address, sigs, err := txscript.ExtractPkScriptAddrs

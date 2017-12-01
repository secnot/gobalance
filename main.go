// Copyright (c) 2014-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"fmt"
	"os"
	"time"
	"os/signal"
	"path/filepath"
	"syscall"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/chaincfg"

	"github.com/secnot/gobalance/crawler"
	"github.com/secnot/gobalance/block_manager"
	"github.com/secnot/gobalance/block_manager/storage"
	"github.com/secnot/gobalance/recent_tx"
	"github.com/secnot/gobalance/balance"
	"github.com/secnot/gobalance/height"
	"github.com/secnot/gobalance/primitives"
	"github.com/secnot/gobalance/api"
	"github.com/secnot/gobalance/config"
)

const (
	DbFilename = "utxo.db"
)

var utxoStorage storage.Storage 

// CleanUp gracefully stop all routines
func CleanUp(vacuum bool) {		
	crawler.Stop()
	block_manager.Stop()

	if utxoStorage != nil {
		if vacuum {
			log.Print("Cleaning Up")
			if err := utxoStorage.CleanUp(); err != nil {
				log.Print(err)
			}
		}
		utxoStorage.Close()
	}
}


// Catch ctrl-c and exit gracefully
func ExitHandler(c chan os.Signal) {
	for sig := range c {
		CleanUp(false)
		log.Print("Exit: ", sig)
		os.Exit(1)
	}
}

func main() {

	// Load default config
	conf, err := config.LoadConfig()
	if err != nil {
		log.Panic(err)
	}

	// Connect to local bitcoin core RPC server using HTTP POST mode.
	rpcConf := rpcclient.ConnConfig{
		Host:         conf["bitcoind.host"].(string),
		User:         conf["bitcoind.user"].(string),
		Pass:         conf["bitcoind.pass"].(string),
		DisableAutoReconnect: false,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	// Configure bitcoind server parameters
	chain := conf["bitcoind.chain"]
	switch chain {
	case "mainnet":
		primitives.SelectChain(&chaincfg.MainNetParams)	
	case "testnet3":
		primitives.SelectChain(&chaincfg.TestNet3Params)
	default:
		log.Panicf("Unsupported bitcoind.chain %v", chain)
	}

	// Initialize utxo storage 
	dbPath := filepath.Join(conf["workdir"].(string), DbFilename)
	absDbPath, err := filepath.Abs(os.Expand(dbPath, os.Getenv))
	if err != nil {
		log.Panic(err)
	}
	
	utxoStorage, err = storage.NewSQLiteStorage(absDbPath)
	if err != nil {
		log.Panic(err)
	}

	// Launch Crawler
	lastHeight, lastBlockHash, err := utxoStorage.GetLastBlock()
	if err != nil {
		log.Panic(err)
	}

	go crawler.Crawler(rpcConf, uint64(lastHeight+1), lastBlockHash)

	// Launch Block Manager
	updateChan := crawler.Subscribe(10)
	
	go block_manager.Manager(
		utxoStorage, 
		int(conf["utxo_cache_size"].(int64)), 
		uint16(conf["recent_blocks"].(int64)), 
		updateChan,
		conf["sync"].(bool), // Enable sync mode
	)
	

	if !conf["sync"].(bool) {
		// Launch balance cache routine
		go balance.BalanceRoutine(int(conf["balance_cache_size"].(int64)))

		// Launch recent transactions routine
		go recent.RecentTxRoutine(uint16(conf["recent_blocks"].(int64)))

		// Launch height routine
		go height.HeightRoutine()
	}

	// Catch interrupts to exit gracefully
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go ExitHandler(c)

	log.Print("Started")

	// Start crawler
	crawler.Start()

	// Wait until the blockchain is synced
	for {
		time.Sleep(time.Second*10)
		if block_manager.Synced() {
			break
		}
	}
	log.Print("Synced")

	// When in sync mode vacuum DB and exit
	if conf["sync"].(bool) {
		CleanUp(true)
		log.Print("Done")
		os.Exit(1)
	}

	// Launch JSON API
	bind := fmt.Sprint("%v:%v", conf["api.bind"].(string), conf["api.port"].(int64))
	api.StartApi(bind, conf["api.url_prefix"].(string))
}


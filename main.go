// Copyright (c) 2014-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"time"
	"github.com/btcsuite/btcrpcclient"
	"github.com/btcsuite/btcd/chaincfg"

	"github.com/secnot/gobalance/crawler"
	"github.com/secnot/gobalance/balance"
	bstorage "github.com/secnot/gobalance/balance/storage"
	ustorage "github.com/secnot/gobalance/crawler/storage"
	"github.com/secnot/gobalance/logger"
	"github.com/secnot/gobalance/primitives"
)


// TODO: Catch ctrl-c and exit gracefully


func main() {
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	rpcConf := btcrpcclient.ConnConfig{
		Host:         "localhost:8332",
		User:         "secnot",
		Pass:         "12345",
		DisableAutoReconnect: false,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	// Crawler
	primitives.SelectChain(&chaincfg.MainNetParams)
	utxoStorage, err := ustorage.NewSQLiteStorage("./DB_utxo.db")
	if err != nil {
		log.Panic(err)
	}

	//TODO: Load height from db?
	blockCrawler, err := crawler.NewCrawler(rpcConf, 0, utxoStorage)
	if err != nil {
		log.Panic(err)
	}

	// Balance
	balanceStorage, err := bstorage.NewSQLiteStorage("./DB_balance.db")
	if err != nil {
		log.Panic(err)
	}
	balanceProc := balance.NewBalanceProcessor(balanceStorage, 200000)
	blockCrawler.Subscribe(balanceProc)

	// Logging
	logBlocks := logger.NewLogger()
	blockCrawler.Subscribe(logBlocks)

	blockCrawler.Start()

	// TODO: Subscribe balance and other services
	for {
		time.Sleep(10000*time.Second)
	}
}


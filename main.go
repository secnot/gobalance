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
	"github.com/secnot/gobalance/balance/storage"
	"github.com/secnot/gobalance/logger"
	"github.com/secnot/gobalance/primitives"
)




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

	primitives.SelectChain(&chaincfg.MainNetParams)
	blockCrawler := crawler.NewCrawler(rpcConf, 0) //TODO: Load height from db

	// Balance
	sqlStorage, err := storage.NewSQLiteStorage("./DB_balance.db")
	if err != nil {
		log.Panic(err)
	}
	balanceProc := balance.NewBalanceProcessor(sqlStorage, 200000)
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


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
	_ "github.com/secnot/gobalance/balance"
	_  "github.com/secnot/gobalance/balance/storage"
	"github.com/secnot/gobalance/primitives"
)




func main() {
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	connCfg := &btcrpcclient.ConnConfig{
		Host:         "localhost:8332",
		User:         "secnot",
		Pass:         "12345",
		DisableAutoReconnect: false,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	//defer client.Shutdown()


	primitives.SelectChain(&chaincfg.MainNetParams)
	blockCrawler := crawler.NewCrawler(client, 0)

	// Balance
	//memStorage := storage.NewMemoryStorage()
	//balanceProc := balance.NewBalanceProcessor(memStorage, 200000) // cachesize
	//blockCrawler.Subscribe(balanceProc)

	// Logging
	blockCrawler.Subscribe(crawler.NewLogger())

	blockCrawler.Start()

	// TODO: Subscribe balance and other services
	for {
		time.Sleep(10000*time.Second)
	}
}


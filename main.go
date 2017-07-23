// Copyright (c) 2014-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"time"
	"github.com/btcsuite/btcrpcclient"
	"github.com/secnot/gobalance/crawler"
	"github.com/btcsuite/btcd/chaincfg"

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
//	craw := crawler.NewCrawler(client, 476800)

//	craw := crawler.NewCrawler(client, 140930)
	craw := crawler.NewCrawler(client, 0)
	confirmed := crawler.NewConfirmedAdapter(6)
	craw.Subscribe(confirmed)
	confirmed.Subscribe(crawler.NewLogger())
	craw.Start()

	// TODO: Subscribe balance and other services
	for {
		time.Sleep(10000*time.Second)
	}
}


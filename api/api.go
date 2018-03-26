package api

import (
	"log"
	"net/http"
	"github.com/secnot/gobalance/balance"
	"github.com/secnot/gobalance/recent_tx"
	"github.com/secnot/gobalance/height"
)

func StartApi(address string, urlPrefix string, 
	balanceC  *balance.BalanceCache, 
	recentTxC *recent_tx.RecentTxCache,
	heightC   *height.HeightCache) {

	router := NewRouter(urlPrefix, balanceC, recentTxC, heightC)
	log.Fatal(http.ListenAndServe(address, router))
}

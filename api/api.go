package api

import (
	"log"
	"time"
	"net/http"
	"crypto/tls"
	"github.com/secnot/gobalance/balance"
	"github.com/secnot/gobalance/recent_tx"
	"github.com/secnot/gobalance/height"
)

func StartApi(address string, urlPrefix string, 
	balanceC  *balance.BalanceCache, 
	recentTxC *recent_tx.RecentTxCache,
	heightC   *height.HeightCache) *http.Server {

	router := NewRouter(urlPrefix, balanceC, recentTxC, heightC)

	srv := &http.Server{
		Addr: address,
		Handler: router,
		//Transport: tr,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		//Disable HTTP2
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	
	go func() {
		log.Printf("HttpServer: Listening on %v\n",address)
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("HttpServer: %v", err)
		}
	}()

	return srv
}

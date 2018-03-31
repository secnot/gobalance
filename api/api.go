package api

import (
	"log"
	"time"
	"net/http"
	"crypto/tls"
	"github.com/secnot/gobalance/interfaces"
)


type ApiServer struct {
	server *http.Server
}

func (s *ApiServer) Stop () error{
	return s.server.Shutdown(nil)
}



// Start api starts server and returns http.Server that can be used to stop it with Shutdown
func StartApi(address string, urlPrefix string, 
	balanceC  interfaces.BalanceCache, 
	recentTxC interfaces.RecentTxCache,
	heightC   interfaces.HeightCache) *ApiServer {

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

	return &ApiServer{server: srv}
}

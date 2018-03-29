package peers


import (
	"crypto/tls"
	"net/http"
	"log"
	"time"
)


// 
func LaunchPeerHttpRoutine(address string, commandCh chan *DiscoveryMsg) *http.Server {

	router := NewRouter("", commandCh, false)

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
		log.Printf("PeerHttpServer: Listening on %v",address)
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("PeerHttpServer: %v", err)
		}
	}()

	return srv
}

package balance


import (
	"fmt"
	"time"
	"net"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/secnot/gobalance/block_manager"
	"github.com/secnot/gobalance/peers"
	"github.com/secnot/gobalance/api/common"
	//"github.com/secnot/gobalance/primitives"
)



var ( 
	BalanceRequestChan   = make(chan BalanceRequest, 100)
	BalanceStopChan      = make(chan chan bool)
)




type BalanceRequest struct {
	Address    string
	IP         net.IP
	ResponseCh chan BalanceResponse
}

type BalanceResponse struct {
	balance int64
	err     error
}


// BalanceProxy
type BalanceProxy struct {
	BlockM    *block_manager.BlockManager
	PeerM     *peers.PeerManager
	CacheSize int
	
	cache     *BalanceCache
}

var proxyClient = &http.Client{
	Timeout:    2 * time.Second,
	Transport : &http.Transport{MaxIdleConnsPerHost: 20},
}



func (b *BalanceProxy) Start(){
	
	b.cache   = NewBalanceCache(b.CacheSize, b.BlockM)
	go b.BalanceRoutine()
}


// 
func (b *BalanceProxy) requestProxyBalance(request BalanceRequest) {

	remotePeer, err := b.PeerM.GetPeerPersistent(request.IP.String())
	url := fmt.Sprintf("http://%s/%s", remotePeer, api_common.BalancePath)
	resp, err := proxyClient.Get(url)
	if err != nil {
		request.ResponseCh <- BalanceResponse{balance: -1, err: err}
		return
	}
	defer resp.Body.Close()

	var address api_common.Address
	err = json.NewDecoder(resp.Body).Decode(&address)
	if err != nil {
		request.ResponseCh <- BalanceResponse{balance: -1, err: err}
		return
	}
	
	ioutil.ReadAll(resp.Body) // Exhaust body data
	request.ResponseCh <- BalanceResponse{balance: address.Balance, err: nil}
}


func (b *BalanceProxy) BalanceRoutine() {

	updateChan := block_manager.Subscribe(10)
	
	// When the block manager is commiting a block the balance is proxied from another
	// 
	proxyMode := false

	for {

		select {
		case update := <- updateChan:			
		
			switch update.Class {
			case block_manager.OP_NEWBLOCK:
				b.cache.NewBlock(update.Block)
			case block_manager.OP_BACKTRACK:
				b.cache.Backtrack(update.Block)
			case block_manager.OP_COMMIT:
				proxyMode = true
			case block_manager.OP_COMMIT_DONE:
				proxyMode = false
			}

		case request := <- BalanceRequestChan:
			if !proxyMode {
				request.ResponseCh <- BalanceResponse{balance: b.cache.GetBalance(request.Address), err: nil}
			} else {
				// TODO: limit number of parallel requests??
				go b.requestProxyBalance(request)
			}
	
		case ch := <- BalanceStopChan:
			// TODO: Close all pending requests, and channels????
			ch <- true
			return
		}
		
	}
}

// Request
func GetBalance(address string, ip net.IP) (balance int64, err error) {	
	responseCh := make(chan BalanceResponse)
	BalanceRequestChan <- BalanceRequest{Address: address, ResponseCh: responseCh, IP: ip}
	response :=  <- responseCh
	close(responseCh)
	return response.balance, response.err
}

func (b *BalanceProxy) Stop() {
	confirmationCh := make(chan bool)
	BalanceStopChan <- confirmationCh
	<- confirmationCh
	close(confirmationCh)
}

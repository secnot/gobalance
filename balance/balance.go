package balance


import (
	"fmt"
	"time"
	"net"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/secnot/gobalance/interfaces"
	"github.com/secnot/gobalance/api/common"
)

// Error returned when remote peer balance request failed
type ConnectionError struct {
	msg string // error description
	StatusCode int 
}
func (e ConnectionError) Error() string {return e.msg}

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
type BalanceCache struct {
	BlockM    interfaces.BlockManager
	PeerM     interfaces.PeerManager
	CacheSize int
	
	cache     *Cache
	
	// Control channels
	RequestChan chan BalanceRequest
	StopChan    chan chan bool
}

// 
var proxyClient = &http.Client {
	Timeout:    2 * time.Second,
	Transport : &http.Transport{MaxIdleConnsPerHost: 20},
}


// NewBalanceCache initializes a BalanceCache
func NewBalanceCache(blockM interfaces.BlockManager, 
					peerM interfaces.PeerManager, cacheSize int) *BalanceCache {
	cache := &BalanceCache {
		BlockM:    blockM,
		PeerM :    peerM,
		CacheSize: cacheSize,
	}

	cache.start()
	return cache
}

// Initialize and start proxy
func (b *BalanceCache) start(){
	
	b.cache   = NewCache(b.CacheSize, b.BlockM)
	b.RequestChan   = make(chan BalanceRequest, 100)
	b.StopChan      = make(chan chan bool)
	go b.balanceRoutine()
}

// 
func (b *BalanceCache) requestProxyBalance(request BalanceRequest) {

	remotePeer, err := b.PeerM.GetPeerPersistent(request.IP.String())
	url := fmt.Sprintf("http://%s/%s/%s", remotePeer, api_common.BalancePath, request.Address)
	resp, err := proxyClient.Get(url)
	if err != nil {
		b.PeerM.MarkPeerUnreachable(remotePeer)
		request.ResponseCh <- BalanceResponse{balance: -1, err: err}
		return
	}
	defer resp.Body.Close()

	// Check valid response status code
	if resp.StatusCode != http.StatusOK {
		err := ConnectionError{
			msg: fmt.Sprintf("Connection error while requesting balance from %v", url),
			StatusCode: resp.StatusCode,
		}

		request.ResponseCh <- BalanceResponse{balance: -1, err: err}
		return
	}
	
	// Decode response
	var address api_common.Address
	err = json.NewDecoder(resp.Body).Decode(&address)
	if err != nil {
		request.ResponseCh <- BalanceResponse{balance: -1, err: err}
		return
	}
	
	ioutil.ReadAll(resp.Body) // Exhaust body data
	request.ResponseCh <- BalanceResponse{balance: address.Balance, err: nil}
}

// balanceRoutine handles all incoming requests
func (b *BalanceCache) balanceRoutine() {

	updateChan := b.BlockM.Subscribe(10)
	
	// When the block manager is commiting a block the balance is proxied from another
	// 
	proxyMode := false

	for {

		select {
		case update := <- updateChan:			
		
			switch update.Class {
			case interfaces.OP_NEWBLOCK:
				b.cache.NewBlock(update.Block)
			case interfaces.OP_BACKTRACK:
				b.cache.Backtrack(update.Block)
			case interfaces.OP_COMMIT:
				proxyMode = true
			case interfaces.OP_COMMIT_DONE:
				proxyMode = false
			}

		case request := <- b.RequestChan:
			if !proxyMode {
				request.ResponseCh <- BalanceResponse{balance: b.cache.GetBalance(request.Address), err: nil}
			} else {
				// TODO: limit number of parallel requests??
				go b.requestProxyBalance(request)
			}
	
		case ch := <- b.StopChan:
			ch <- true
			return
		}
	}
}

// GetBalance Request address balance
func (b *BalanceCache) GetBalance(address string, ip net.IP) (balance int64, err error) {	
	responseCh := make(chan BalanceResponse)
	b.RequestChan <- BalanceRequest{Address: address, ResponseCh: responseCh, IP: ip}
	response :=  <- responseCh
	close(responseCh)
	return response.balance, response.err
}

func (b *BalanceCache) Stop() {
	confirmationCh := make(chan bool)
	b.StopChan <- confirmationCh
	<- confirmationCh
	close(confirmationCh)
}

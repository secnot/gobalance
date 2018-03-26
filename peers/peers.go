package peers

import (
	"fmt"
	"time"
	"net"
	"net/http"
	"sync"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"github.com/secnot/gobalance/utils"
)

//var peerClient = &http.Client{Timeout: 2 * time.Second}

var peerClient = &http.Client{
	Timeout:    2 * time.Second,
	Transport : &http.Transport{MaxIdleConnsPerHost: 20},
}

// Peer connection status at the time of the last update
type PeerConnectionStatus int 

const (
	// Perr responded to last status request
	ReachableStatus PeerConnectionStatus = iota

	// The peer was contacted but is now marked as unreachable
	UnreachableStatus

	// The peer was never contacted
	UnknownStatus
)

// Peer modes of operation
type PeerMode string

const (
	// Peer mode is unknown
	UnknownMode PeerMode = "unknown"

	// Peer routes balance request to other peers
	LoadBalanceMode      = "loadbalance"
	
	// Full balance api support
	FullMode             = "full"

	// Just seed for avaialble nodes
	SeedMode             = "seed"
)


type Peer struct {

	sync.RWMutex

	// Listen ip
	ip net.IP

	// Peer protocol listen port
	peerPort uint16

	// Balance api listen port
	apiPort uint16

	// mode of operation
	mode PeerMode

	// Time of the last update
	lastUpdate time.Time

	// Last time the peer was reachable
	lastReachable time.Time

	// Connection retries since the last time peer was reachable
	connectionRetries uint32

	// State of the peer connection in the last update
	connectionStatus PeerConnectionStatus

	// peer software version
	version string

	// flag if the remote peer has discovered this node 
	discovered bool	
}

// NewPeer
func NewPeer(ip net.IP, peerPort uint16) *Peer {

	peer := Peer{
		ip:                ip,
		peerPort:          peerPort,
		apiPort:           0,
		mode:              UnknownMode,
		connectionRetries: 0,
		connectionStatus : UnknownStatus, 
		lastUpdate:        time.Now(),
		lastReachable:     time.Now(),
		version:           "",
		discovered:        false,
	}

	return &peer
}

// Ip returns peer ip
func (p *Peer) Ip() net.IP {
	p.RLock()
	defer p.RUnlock()

	return p.ip
}

// Port returns peer port
func (p *Peer) PeerPort() uint16 {
	p.RLock()
	defer p.RUnlock()

	return p.peerPort
}

// ApiPort getter 
func (p *Peer) ApiPort() uint16 {
	p.RLock()
	defer p.RUnlock()
		
	return p.apiPort
}

// SetApiPort sets apiport
func (p *Peer) SetApiPort(port uint16) {
	p.Lock()
	defer p.Unlock()

	p.apiPort = port
}

// Equal compare to peers
func (p *Peer) Equal(other *Peer) bool {
	return p.Ip().Equal(other.Ip()) && p.PeerPort() == other.PeerPort() && p.ApiPort() == other.ApiPort()
}

// Host returns peer api ip:port string
func (p *Peer) PeerHost() string {
	return utils.HostToString(p.Ip(), p.PeerPort())
}

// ApiHost returns balance api ip:port string (or "" if apiPort is not available)
func (p *Peer) ApiHost() string {
	return utils.HostToString(p.Ip(), p.ApiPort())
}

// Url is a helper to build peer protocol urls
func (p *Peer) Url(path string) string{
	return fmt.Sprintf("http://%s/%s", p.PeerHost(), path)
}

// Mode getter
func (p *Peer) Mode() PeerMode {
	p.RLock()
	defer p.RUnlock()

	return p.mode
}

// SetMode (setter)
func (p *Peer) SetMode(mode PeerMode) {
	p.Lock()
	defer p.Unlock()

	p.mode = mode
}

// ConnectionStatus returns peers connection status
func (p *Peer) ConnectionStatus() PeerConnectionStatus {
	p.RLock()
	defer p.RUnlock()

	return p.connectionStatus
}

// ConnectionRetries returns the number of times the connection failed
func (p *Peer) ConnectionRetries() uint32 {
	p.RLock()
	defer p.RUnlock()

	return p.connectionRetries
}

// LastUpdate
func (p *Peer) LastUpdate() time.Time {
	p.RLock()
	defer p.RUnlock()

	return p.lastUpdate
}

// LastReachable
func (p *Peer) LastReachable() time.Time {
	p.RLock()
	defer p.RUnlock()

	return p.lastReachable
}

// Reachable
func (p *Peer) SetReachable() {
	p.Lock()
	defer p.Unlock()
	
	p.connectionStatus  = ReachableStatus
	p.connectionRetries = 0
	p.lastUpdate = time.Now()
	p.lastReachable = time.Now()

}


func (p *Peer) SetUnreachable() {
	p.Lock()
	defer p.Unlock()

	if p.connectionStatus == ReachableStatus {
		p.connectionStatus = UnreachableStatus
	}

	p.connectionRetries += 1
	p.lastUpdate = time.Now()
}

// Reachable returns true if the peer was reachable during last update
func (p *Peer) Reachable() bool {
	return p.ConnectionStatus() == ReachableStatus
}

// Discovered setter and getter
func (p *Peer) SetDiscovered(flag bool) {
	p.Lock()
	defer p.Unlock()

	p.discovered = flag
}

func (p *Peer) Discovered() bool {
	p.RLock()
	defer p.RUnlock()

	return p.discovered
}

// SetVersion version setter
func (p *Peer) SetVersion(version string) {
	p.Lock()
	defer p.Unlock()

	p.version = version
}

// Version getter
func (p *Peer) Version() string {
	p.RLock()
	defer p.RUnlock()
	return p.version
}

// SetFromStatus Set Peer fields from status response
func (p *Peer) SetFromStatus(s Status) {
	p.Lock()
	defer p.Unlock()

	p.mode = s.Mode
	p.peerPort = s.Port
	p.apiPort = s.BalancePort
	p.version = s.Version
}

// CheckStatus connects to peer to get current status
func (p *Peer) RequestStatus() (Status, error) {
	resp, err := peerClient.Get(p.Url(StatusPath))
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()

	var status Status
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return Status{}, err
	}

    ioutil.ReadAll(resp.Body)
	return status, nil
}

// PeerList obtains peer's peer list
func (p *Peer) RequestPeerList() ([]string, error) {
	
	resp, err := peerClient.Get(p.Url(PeerListPath))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var peers []string
	err = json.NewDecoder(resp.Body).Decode(&peers)
	if err != nil {
		return nil, err
	}

    ioutil.ReadAll(resp.Body) // 
	return peers, nil
}

// Announce a new host to the peer
func (p *Peer) RequestAnnouncePeer(status Status) error {
	
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	
	resp, err := peerClient.Post(p.Url(AnnouncePath), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	return nil
}

/*
func (p *Peer) RequestAnnouncePeer(status Status) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
    
	req, err := http.NewRequest("POST", p.Url(AnnouncePath), bytes.NewBuffer(data))
    //req.Header.Set("X-Custom-Header", "myvalue")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }

    defer resp.Body.Close()
    ioutil.ReadAll(resp.Body)
	return nil
}
*/

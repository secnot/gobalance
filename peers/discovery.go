package peers

import (
	"fmt"
	"log"
	"time"
	"net"
	"context"
	"net/http"
	"github.com/secnot/gobalance/utils"
	"github.com/secnot/orderedmap"
)

const (
	// Period between announcements
	defaultAnnouncementPeriod = 5 * time.Second
	
	// Period between peer list requests
	defaultPeerDiscoveryPeriod = 10 * time.Second

	// period between status updates
	defaultStatusUpdatePeriod = 5 * time.Second

	// Number of retries before a peer is discarded
	defaultPeerMaxRetries = 10
)

type discoveryMsgType int

const (
	UnreachablePeerMsg  discoveryMsgType = iota
	
	// Peer pointer in data
	AddPeerMsg

	// Delete peer from handler
	DeletePeerMsg

	// Request for the reachable peers list (channel for the response with data)
	PeerListRequestMsg

	// Request for the peer status (channel for the response with data)
	StatusRequestMsg
	
	// Close peer handler and its routines
	StopHandlerMsg

	// Announcement from another peer (AnnouncementData struct include with data)
	PeerAnnouncementMsg
)

func (d discoveryMsgType) String() string{

	switch d {
		case UnreachablePeerMsg:
			return "UnreachablePeerMsg"
		case AddPeerMsg:
			return "AddPeerMsg"
		case DeletePeerMsg:
			return "DeletePeerMsg"
		case StopHandlerMsg:
			return "StopHandlerMsg"
	}
	return ""
}


// Discovery routine message format
type DiscoveryMsg struct {
	typ discoveryMsgType
	data interface{}
}

func NewDiscoveryMsg(typ discoveryMsgType, data interface{}) *DiscoveryMsg {
	c := DiscoveryMsg {
		typ: typ,
		data: data,
	}

	return &c
}

func (d DiscoveryMsg) String() string{
	return fmt.Sprintf("%v: %v", d.typ, d.data)
}


// Peer status message
type PeerStatusMsg struct {
	hostname string
	reachable bool
	status Status
}



type PeerHandler struct {
	
	// Configuration options
	Mode         PeerMode
	PeerPort     uint16
	BalancePort  uint16
	Version      string
	SeedNodes    []string
	AllowLocalIP bool

	// Channel used to send updates about available peers
	subscriberCh chan *DiscoveryMsg

	// Channel used to receive command (unreachable, add, delete, ...)
	commandCh    chan *DiscoveryMsg
	
	// Min time between announcements to new peers
	AnnouncementPeriod  time.Duration

	// Time between requests for new peers
	PeerDiscoveryPeriod time.Duration
	
	// Peer status update period
	StatusUpdatePeriod  time.Duration

	// Max unsuccessful retries until a peer is discarded
	PeerMaxRetries      uint32

	// when the handler was launched
	startTime time.Time

	// Seed nodes with unresolved hostnames
	manager *PeerManager

	// All the peers discovered for now 
	peers *orderedmap.OrderedMap

	// apiHost string to its equivalent peerHost string
	apiHostIdx map[string] string

	// Channel used to communicate with resolveHostnameRoutine
	resolveRequestCh  chan string
	resolveResponseCh chan string
	resolveCloseCh    chan bool

	// Channel for status updates from status routines
	statusUpdateCh    chan PeerStatusMsg

	// Slice with all the avaialble local IPs
	localIps []net.IP

	// Pointer to http server handling peer requests (for closing on exit)
	httpServer *http.Server

	// Started flag
	started bool
}

// Start, initialize, and lauch PeerHandler
// TODO: Return error when the port is in use
func (p *PeerHandler) Start() (updateCh chan *DiscoveryMsg){

	// Set defaults for missing timmings
	if p.AnnouncementPeriod  == 0 {
		p.AnnouncementPeriod = defaultAnnouncementPeriod
	}
	if p.PeerDiscoveryPeriod == 0 {
		p.PeerDiscoveryPeriod = defaultPeerDiscoveryPeriod
	}
	if p.StatusUpdatePeriod == 0 {
		p.StatusUpdatePeriod = defaultStatusUpdatePeriod
	}
	if p.PeerMaxRetries == 0 {
		p.PeerMaxRetries = defaultPeerMaxRetries
	}

	// Channels to send updates and receive commands
	p.subscriberCh     = make(chan *DiscoveryMsg, 100)
	p.commandCh        = make(chan *DiscoveryMsg, 100)

	// 
	p.peers             = orderedmap.NewOrderedMap()
	p.apiHostIdx        = make(map[string]string)
	p.resolveRequestCh  = make(chan string, 1000)
	p.resolveResponseCh = make(chan string, 10)
	p.resolveCloseCh    = make(chan bool)
	p.statusUpdateCh    = make(chan PeerStatusMsg, 10)
	p.startTime         = time.Now()

	// Extract list of host ips
	p.localIps = utils.GetLocalIPs(true)
	
	// Lauch http server.
	address := fmt.Sprintf(":%v", p.PeerPort)
	p.httpServer = LaunchPeerHttpRoutine(address, p.commandCh) 

	// Start resolve hostname routine and queue seed hostnames.
	go resolveHostnameRoutine(p.resolveRequestCh, p.resolveResponseCh, p.resolveCloseCh)
	utils.Shuffle(p.SeedNodes, utils.NewRandSource())
	for _, seed := range p.SeedNodes {
		p.resolveRequestCh <- seed
	}

	// DiscoveryRoutine
	go p.peerDiscoveryRoutine()

	// Mark handler as started
	p.started = true
	return p.subscriberCh
}

// Stop peerhandler and its routines
func (p *PeerHandler) Stop() {
	p.commandCh <- NewDiscoveryMsg(StopHandlerMsg, nil)
}

// Mark one of the peers unreachable
func (p *PeerHandler) MarkPeerUnreachable(hostname string) {
	p.commandCh <- NewDiscoveryMsg(UnreachablePeerMsg, hostname)
}

// PeerDiscoveryRoutine
func (p *PeerHandler) peerDiscoveryRoutine() {
	
	resolveResponseCh := p.resolveResponseCh
	statusUpdateCh    := p.statusUpdateCh
	commandCh         := p.commandCh

	statusTicker   := time.NewTicker(p.StatusUpdatePeriod)  // Leaks
	announceTicker := time.NewTicker(p.AnnouncementPeriod)  // Leaks
	listTicker     := time.NewTicker(p.PeerDiscoveryPeriod) // Leaks

	
	for {

		select {

		// Messages from discovery routines
		case msg := <-commandCh:
		
			switch msg.typ {
			case UnreachablePeerMsg:
				if peerHost, ok := p.apiHostIdx[msg.data.(string)]; ok {
					p.markPeerUnreachable(peerHost)
				}
			
			case StopHandlerMsg:
				if !p.started {
					return
				}
				statusTicker.Stop()
				listTicker.Stop()
				announceTicker.Stop()
				p.shutdown()
				p.started = false
				// TODO: Close channels
				return

			case PeerListRequestMsg:
				responseCh := msg.data.(chan []string)
				responseCh <- p.getPeerList()

			case StatusRequestMsg:
				responseCh := msg.data.(chan Status)
				responseCh <- p.getStatus()

			case PeerAnnouncementMsg:
				data := msg.data.(AnnouncementData)

				// Add if it doesn't exists
				newPeer := NewPeer(data.ip, data.status.Port)
				peer, ok := p.getPeer(newPeer.PeerHost()); 
				if !ok {
					p.addPeer(newPeer)
					peer = newPeer
				}
				peer.SetDiscovered(true)
				log.Printf("Peer Announcement: %v %v\n", peer.PeerHost(), data.status.Mode)
				
				// Handle status as normal status update
				statusUpdateCh <- PeerStatusMsg{hostname: peer.PeerHost(), reachable: true, status:data.status,}
			default:
				log.Panic(msg)

			}
		// New peer hostname discovered from a list request and subsequentially 
		// sent to be resolved.
		case hostname := <- resolveResponseCh:
			// Only add resolved peer if it doesn't already exists
			if _, ok := p.getPeer(hostname); ok {
				continue
			}
		
			ip, port, err := utils.ParseHost(hostname)
			if err != nil {
				continue
			}

			// Check the ip is not one of the local ips (if enabled)
			if p.AllowLocalIP || !p.isLocalIp(ip) {
				log.Printf("Peer Discovered: %v\n", hostname)
				p.addPeer(NewPeer(ip, port))
			}

		// Start a peer list request
		case <-listTicker.C:
			peer := p.getNextListPeer()
			if peer != nil {
				go getPeerListRoutine(peer, p.resolveRequestCh)
			}

		// Start a peer status request
		case <-statusTicker.C:
			peer := p.getNextStatusUpdatePeer()
			if peer != nil {
				go getPeerStatusRoutine(peer, p.statusUpdateCh)
			}

		// Start peer announce 
		case <-announceTicker.C:
			peer := p.getNextAnnouncePeer()
			if peer != nil {
				go announcePeerRoutine(peer, p.getStatus())
			}

		// Received peer status update from status routine
		case msg := <- statusUpdateCh:
			if msg.reachable {
				p.setPeerStatus(msg.hostname, msg.status)
			} else {
				p.markPeerUnreachable(msg.hostname)
			}

		}
	}
}

// isLocalIp returns true if the ip is one of the host ips
func (p *PeerHandler) isLocalIp(ip net.IP) bool {		
	
	for _, local := range p.localIps {
		if local.Equal(ip) {
			return true
		}
	}
	return false
}

// GetStatus
func (p *PeerHandler) getStatus() Status {
	return Status{
		Mode: p.Mode,
		Port: p.PeerPort,
		BalancePort: p.BalancePort,
		Uptime: int64(time.Since(p.startTime).Seconds()),
		Version: p.Version,
	}
}

// GetPeerList returns the list of currently active peers
func (p *PeerHandler) getPeerList() []string {
	reachable := make([]string, 0, p.peers.Len())
	iter := p.peers.Iter()
	for _, peer, ok := iter.Next(); ok; _, peer, ok = iter.Next() {
		if peer.(*Peer).Reachable() {
			reachable = append(reachable, peer.(*Peer).PeerHost())
		}
	}

	return reachable
}

// Close and stop handler routines
func (p *PeerHandler) shutdown() {
	p.resolveCloseCh <- true	
	if err := p.httpServer.Shutdown(context.Background()); err != nil {
		log.Panic(err)
	}
}

// addPeer add or substitute peer
func (p *PeerHandler) addPeer(peer *Peer) {
	// Delete peer if it already exist
	if _, ok := p.getPeer(peer.PeerHost()); ok {
		p.delPeer(peer.PeerHost())
	}
	p.peers.Set(peer.PeerHost(), peer)
}

// delPeer delete an existing peer by PeerHost address
func (p *PeerHandler) delPeer(addr string) {

	peer, ok := p.getPeer(addr)
	if !ok {
		return
	}
	
	delete(p.apiHostIdx, peer.ApiHost())
	p.peers.Delete(addr)

	if peer.Reachable() && peer.Mode() == FullMode {
		p.subscriberCh <- NewDiscoveryMsg(DeletePeerMsg, peer.ApiHost())
	}
}

// markPeerUnreachable
// signal-> when true signal manager when there is a status change
func (p *PeerHandler) markPeerUnreachable(addr string) {

	peer, ok := p.getPeer(addr)
	if !ok {
		return
	}

	if peer.Reachable() && peer.Mode() == FullMode {
		p.subscriberCh <- NewDiscoveryMsg(DeletePeerMsg, peer.ApiHost())
	}
	peer.SetUnreachable()

	// Delete peers with too many retries
	if peer.ConnectionRetries() > p.PeerMaxRetries {
		p.delPeer(addr)
	}
}

// setPeerStatus
func (p *PeerHandler) setPeerStatus(addr string, status Status) {

	if peer, ok := p.getPeer(addr); ok {
		originalReachable := peer.Reachable()

		ip, port, err := utils.ParseHost(addr)
		if err != nil {
			return
		}

		// Delete old peer if the api port changed
		tempPeer := NewPeer(ip, port)
		tempPeer.SetFromStatus(status)
		if !tempPeer.Equal(peer) {
			p.addPeer(tempPeer) // This will delete old peer
			peer = tempPeer
		}
	
		// Update status and mark reachable
		peer.SetFromStatus(status)
		peer.SetReachable()
		p.apiHostIdx[peer.ApiHost()] = peer.PeerHost()
		
		// If original wasn't reachable or was deleted signal subscribers with new peer	
		if !originalReachable || tempPeer == peer {
			if peer.Mode() == FullMode {
				p.subscriberCh <- NewDiscoveryMsg(AddPeerMsg, peer.ApiHost())
			}
		}
	}
}



/*
 * 	PEER SELECTION FUNCTIONS
 */

// getPeer returns peer if it exists
func (p *PeerHandler) getPeer(hostname string) (peer *Peer, ok bool) {
	apeer, ok := p.peers.Get(hostname)
	if ok {
		return apeer.(*Peer), true
	}

	return nil, false
}

// receives to peer pointers (either can be nil) and returns the selected one (or nil)
type peerSelector func(peer1 *Peer, peer2 *Peer) *Peer

// receives a peer pointer and returns true if it is a valid selection
type firstPeerSelector func(peer *Peer) bool

// iterater through all the peers comparing them in pairs and returns the last selection
func (p *PeerHandler) selectPeer(selector peerSelector) *Peer {
	
	var selection *Peer = nil

	iter := p.peers.Iter()
	for _, apeer, ok := iter.Next(); ok; _, apeer, ok = iter.Next() {
		selection = selector(selection, apeer.(*Peer))	
	}

	if selection != nil {
		p.peers.MoveLast(selection.PeerHost())
	}

	return selection
}

// returns first peer accepted by selection function
func (p *PeerHandler) selectFirstPeer(selector firstPeerSelector) *Peer {

	var selection *Peer = nil 

	iter := p.peers.Iter()
	for _, apeer, ok := iter.Next(); ok; _, apeer, ok = iter.Next() {
		if selector(apeer.(*Peer)) {
			selection =  apeer.(*Peer)
			break
		}
	}

	if selection != nil {
		p.peers.MoveLast(selection.PeerHost())
	}

	return selection
}

// first selector for reachable peers
func reachableSelector(peer *Peer) bool {
	return peer.Reachable()
}

// first selector for seed peers
func seedSelector(peer *Peer) bool {
	return peer.Mode() == SeedMode
}

// first selector for peers that haven't discovered this one
func undiscoveredSelector(peer *Peer) bool {
	return !peer.Discovered()
}

// first selector for peers with unknown status
func unknownConnectionSelector(peer *Peer) bool {
	return peer.ConnectionStatus() == UnknownStatus
}

// first selector for first peer
func firstAvailableSelector(peer *Peer) bool {
	return true
}

// getNextListPeer returns the next peer to be queried for its peer list
func (p *PeerHandler) getNextListPeer() *Peer {
	// First try for a seed node.
	if peer := p.selectFirstPeer(seedSelector); peer != nil {
		return peer
	}

	// If there are no seed nodes available try first reachable node
	return p.selectFirstPeer(reachableSelector)
}

func (p *PeerHandler) getNextStatusUpdatePeer() *Peer {

	// First try a peer in unknown connectionStatus (never connected)
	if peer := p.selectFirstPeer(unknownConnectionSelector); peer != nil {
		return peer
	}

	// 
	return p.selectFirstPeer(firstAvailableSelector)
}

func (p *PeerHandler) getNextAnnouncePeer() *Peer {
	return p.selectFirstPeer(undiscoveredSelector)
}

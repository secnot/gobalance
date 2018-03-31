package peers

import (
	//"fmt"
	"log"
	"sync"
	"time"
	"errors"
	"github.com/secnot/simplelru"
	"github.com/secnot/gobalance/primitives/queue"
	"github.com/secnot/orderedmap"
)

var ErrNoPeersAvailable = errors.New("Backtrack limit reached")


const (
	// Number of unreachable marks required to mark peer unreachable
	DefaultUnreachableMarks = 3

	//
	DefaultUnreachablePeriod = 30*time.Second

	//
	DefaultClientPeerCacheSize = 50000
	DefaultClientPeerPruneSize = 1000
)

type PeerManager struct {
	
	sync.RWMutex
	
	// node mode of opertion 
	Mode PeerMode

	// Port used by peer protocol
	PeerPort uint16

	// Port used by balance api
	BalancePort uint16

	// software version 
	Version string

	// Allow/Disallow peers using a local ip (mainly for testing)
	AllowLocalIP bool
		
	// Peer timmings if not set will use default values (same meaning as PeerHandler)
	AnnouncementPeriod  time.Duration
	PeerDiscoveryPeriod time.Duration
	StatusUpdatePeriod  time.Duration
	PeerMaxRetries      uint32

	// Number of unreachable marks required to mark peer unreachable
	UnreachableMarks uint32

	// 
	UnreachablePeriod time.Duration

	//
	ClientPeerCacheSize int

	// Initial seed peers (format "hostname:port" / "ip:port"
	Seeds []string

	// peer -> unreachable marks queue
	peers *orderedmap.OrderedMap

	// Cache that remembers the peer assigned to each client so it's
	// possible to return the same peer to sucessive requests by the 
	// same client (if the peer is still available)
	clientPeerCache *simplelru.LRUCache

	// Channel used by peer routine to send updates on the available peers
	updateCh  chan *DiscoveryMsg

	// Stop signal channel
	stopCh chan bool

	// Peer handler for 
	handler *PeerHandler

	// Flag indicating the manages has started
	started bool
}

// NewPeerManager
func (p *PeerManager) Start() error {
	
	seeds := make([]string, len(p.Seeds))
	copy(seeds, p.Seeds)

	handler := &PeerHandler {
		Mode:                p.Mode,
		PeerPort:            p.PeerPort,
		BalancePort:         p.BalancePort,
		Version:             p.Version,
		SeedNodes:           seeds,
		AllowLocalIP:        p.AllowLocalIP,
		AnnouncementPeriod:  p.AnnouncementPeriod,
		PeerDiscoveryPeriod: p.PeerDiscoveryPeriod,
		StatusUpdatePeriod:  p.StatusUpdatePeriod,
		PeerMaxRetries:      p.PeerMaxRetries,
	}

	// Load defaults for missing options
	if p.UnreachableMarks == 0 {
		p.UnreachableMarks = DefaultUnreachableMarks
	}
	if p.UnreachablePeriod == 0 {
		p.UnreachablePeriod = DefaultUnreachablePeriod
	}
	if p.ClientPeerCacheSize < 1 {
		p.ClientPeerCacheSize = DefaultClientPeerCacheSize
	}
	

	//TODO: Return error when port was in use
	updateCh := handler.Start()

	p.clientPeerCache = simplelru.NewLRUCache(
		p.ClientPeerCacheSize,
		DefaultClientPeerPruneSize)
	p.peers    = orderedmap.NewOrderedMap()
	p.handler  = handler
	p.updateCh = updateCh
	p.stopCh   = make(chan bool)
	p.started  = true

	go p.peerUpdateRoutine()
	return nil
}


// Stop peer manager 
func (p *PeerManager) Stop() {
	p.Lock()
	if !p.started {
		return
		p.Unlock()
	}
	stopCh := p.stopCh
	p.Unlock()
	
	stopCh <- true
}

// PeerUpdateRoutine
func (p *PeerManager) peerUpdateRoutine() {
	
	p.Lock()
	updateCh := p.updateCh
	stopCh   := p.stopCh
	p.Unlock()

	for {
		select {
		case msg := <- updateCh:
			switch msg.typ {
			case AddPeerMsg:
				p.addPeer(msg.data.(string))
			
			case DeletePeerMsg:
				p.delPeer(msg.data.(string))

			default:
				log.Print(msg.typ)
			}

		case <-stopCh:
			p.Lock()
			p.started = false
			p.peers = orderedmap.NewOrderedMap()
			p.handler.Stop()
			p.Unlock()
			return
		}
	}
}

// GetPeer returns address (ip:port) of one of the peers (round robin)
func (p *PeerManager) getPeer() (string, error) {
	if !p.started {
		return "", ErrNoPeersAvailable
	}

	peer, _, ok := p.peers.GetFirst()
	if !ok {
		return "", ErrNoPeersAvailable
	}

	p.peers.MoveLast(peer.(string))
	return peer.(string), nil
}

// GetPeer returns address (ip:port) of one of the peers (round robin)
func (p *PeerManager) GetPeer() (string, error) {
	p.Lock()
	defer p.Unlock()
	return p.getPeer()
}

// GetPeerPersistent return  the address (ip:port) of one of the peers, but
// tries to return the same peer to calls with the same id. 
// (while available and not purged from cache)
func (p *PeerManager) GetPeerPersistent(id string) (string, error) {
	p.Lock()
	defer p.Unlock()

	// try for cache id
	if peer, ok := p.clientPeerCache.Get(id); ok {
		// Client peer is cached check it is available
		if _, ok := p.peers.Get(peer.(string)); ok {
			return peer.(string), nil
		}
	}
	
	// Assign new peer to client
	peer, err := p.getPeer()
	if err == nil {
		p.clientPeerCache.Set(id, peer)
	}
	return peer, err
}

// MarkPeerUnreachable
func (p *PeerManager) MarkPeerUnreachable(peer string) {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		return
	}

	// If the peer was already deleted exit
	marks, ok := p.peers.Get(peer)
	if !ok {
		return
	}
	
	addUnreachableMark(marks.(*queue.Queue), p.UnreachablePeriod)

	if isUnreachable(marks.(*queue.Queue), p.UnreachableMarks) {
		p.peers.Delete(peer)
		p.handler.MarkPeerUnreachable(peer)
	}
}

// addUnreachableMark adds a new unreachable mark and deletes expired ones,
// to and from peer.
func addUnreachableMark(marks *queue.Queue, period time.Duration) {

	now := time.Now()
	marks.PushBack(now)

	// Remove old marks
	for marks.Len() > 0 {
		markTime := marks.Front().(time.Time)
		since := now.Sub(markTime)
		if since > period {
			marks.PopFront()
		} else {
			break
		}
	}
}

func isUnreachable(marks *queue.Queue, maxMarks uint32) bool {
	return uint32(marks.Len()) > maxMarks
}

// addPeer used by peer discovery routine to add peers
func (p *PeerManager) addPeer(peer string) {
	p.Lock()
	defer p.Unlock()

	p.peers.Set(peer, queue.New())
}

// delPeer used by peer discovery routine to remove peers
func (p *PeerManager) delPeer(peer string) {
	p.Lock()
	defer p.Unlock()

	p.peers.Delete(peer)
}



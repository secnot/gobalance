package mocks

import (
	"sync"
)


// Peer manager mock using PeerManager interface
type PeerMock struct {
	sync.RWMutex
	// Address returned everytime a peer is requested
	Address string
}

// Not implemented
func (p *PeerMock) Start() error {
	return nil
}
func (p *PeerMock) Stop() {
	return
}
func (p *PeerMock) MarkPeerUnreachable(peer string) {
	return
}

// SetPeer change returned peer address
func (p *PeerMock) SetPeer(peerAddress string) {
	p.Lock()
	defer p.Unlock()
	p.Address = peerAddress
}

// GetPeer returns peer address
func (p *PeerMock) GetPeer()(string, error) {
	p.RLock()
	defer p.RUnlock()
	return p.Address, nil
}

// GetPerrPersistent returns peer address
func (p *PeerMock) GetPeerPersistent(id string)(string, error) {
	p.RLock()
	defer p.RUnlock()
	return p.Address, nil
}

// NewPeerMock returns initialized PeerMock
func NewPeerMock(peerAddress string) *PeerMock {
	mock := &PeerMock {
		Address: peerAddress,
	}

	return mock
}



package balance

import (
	"sync"
	"github.com/secnot/gobalance/interfaces"
)



// Mock Block Manager
//////////////////////

// Block manager mock using BlockManagerInterface
type BlockMock struct {

	sync.RWMutex
	balance map[string]int64
	IsSynced bool
	Height int64
	subscribers map[interfaces.UpdateChan] bool
}

// Subscribe to new block updates
func (b *BlockMock) Subscribe(chanSize uint) interfaces.UpdateChan {
	
	b.Lock()
	defer b.Unlock()
	ch := make(interfaces.UpdateChan, 10)
	b.subscribers[ch] = true
	return ch
}

// Cancel subscription (NOT IMPLEMENTED)
func (b *BlockMock) Unsubscribe(ch interfaces.UpdateChan) {
	b.Lock()
	defer b.Unlock()
	delete(b.subscribers, ch)
}

// Return address balance
func (b *BlockMock) GetBalance(address string) int64 {
	b.RLock()
	defer b.RUnlock()
	if balance, ok := b.balance[address]; ok {
		return balance
	} 
	return 0
}

// Set address balance
func (b *BlockMock) SetBalance(address string, balance int64) {
	b.Lock()
	defer b.Unlock()
	b.balance[address] = balance
}

// Get current blockchain height
func (b *BlockMock) GetHeight() (height int64) {
	b.RLock()
	defer b.RUnlock()
	return b.Height
}

// Set current blockchahin height
func (b *BlockMock) SetHeight(height int64) {
	b.Lock()
	defer b.Unlock()
	b.Height = height
}

// Return true if manager synced with bitcoind
func (b *BlockMock) Synced() (sync bool) {
	b.RLock()
	defer b.RUnlock()
	return b.IsSynced
}

// Safely stop block manager
func (b *BlockMock) Stop() {
	return
}

// Method to send a block update to subs 
func (b *BlockMock) SendBlockUpdate(update interfaces.BlockUpdate){
	b.RLock()
	defer b.RUnlock()
	for subscriber, _ := range b.subscribers {
		subscriber <- update
	}
}

// NewBlockMock returns initialized BlockMock
func NewBlockMock(balances map[string]int64, height int64, synced bool) (*BlockMock) {

	mock := &BlockMock {
		balance:  make(map[string]int64),
		IsSynced: synced,
		Height:   height,
		subscribers: make(map[interfaces.UpdateChan]bool),
	}

	for address, balance := range balances {
		mock.SetBalance(address, balance)
	}

	return mock
}



// Mock Peer Manager
/////////////////////

// Peer manager mock using PeerManagerInterface
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



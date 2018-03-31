package interfaces

// Peer Manager interface only purpose is testing
type PeerManager interface {
	Start() error
	Stop()
	MarkPeerUnreachable(peer string)
	GetPeer()(string, error)
	GetPeerPersistent(id string)(string, error)
}


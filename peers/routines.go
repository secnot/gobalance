package peers


import (
	"log"
)


// PeerListRoutine
// peer -> peer to query
// resultCh -> channnel where all the returned peer hostnames are sent on success
func getPeerListRoutine(peer *Peer, resultCh chan string) {
	peerList, err := peer.RequestPeerList()
	if err != nil {
		return
	}

	for _, hostname := range peerList {
		resultCh <- hostname
	}
}


// getPeerStatusRoutine
// peer ->
// statusCh ->
func getPeerStatusRoutine(peer *Peer, statusCh chan PeerStatusMsg) {

	status, err := peer.RequestStatus()
	if err != nil {
		statusCh <- PeerStatusMsg{hostname:peer.PeerHost(), reachable: false}
	} else {
		statusCh <- PeerStatusMsg{hostname: peer.PeerHost(), reachable: true, status:status,}
	}
}


// announcePeerRoutine Send status announcement to peer
func announcePeerRoutine(peer *Peer, status Status) {
	err := peer.RequestAnnouncePeer(status)
	if err != nil {
		log.Printf("announcePeerRoutine: %v\n", err)
	} else {
		peer.SetDiscovered(true)
	}
}

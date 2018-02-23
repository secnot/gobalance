package peers


import (
	"testing"
	"time"
)


func CreatePeerManager(peerPort, balancePort uint16, seeds []string, mode PeerMode) *PeerManager {

	seedsCopy := make([]string, len(seeds))
	copy(seedsCopy, seeds)

	manager := &PeerManager {
		Mode:         mode,
		PeerPort:     peerPort,
		BalancePort:  balancePort,
		Version:      "something something",
		Seeds:        seedsCopy,
		AllowLocalIP: true,
	
		UnreachableMarks: 2, 
		UnreachablePeriod: time.Millisecond,


		// Fast testing
		AnnouncementPeriod:  time.Millisecond*50,
		PeerDiscoveryPeriod: time.Millisecond*50,
		StatusUpdatePeriod:  time.Millisecond*50,
		PeerMaxRetries:      3,
	}

	return manager
}


// GetPeerManagerAvilablePeers returns a map containing all 
func GetPeerManagerAvailablePeers(manager *PeerManager) map[string]bool {

	seen := make(map[string]uint)

	for {
		peer, err  := manager.GetPeer()
		if err != nil {
			break
		}

		if count, ok := seen[peer]; ok {
			// update counte
			if count > 2 {
				break
			} else {
				seen[peer] = count + 1 
			}
		} else {
			seen[peer] = 1
		}
	}

	// copy results to a map[string]bool
	result := make(map[string]bool)
	for key, _ := range seen {
		result[key] = true
	}

	return result
}


// PeerPropagationHelper
func PeerManagerPropagationHelper(t *testing.T, 
		manager1, manager2, manager3 *PeerManager, 
		discovered1, discovered2, discovered3 []string, 
		event func(t *testing.T), eventDelay, stopDelay time.Duration) {
	
	manager1.Start()
	manager2.Start()
	manager3.Start()

	// Stop signal to stop peers
	stopTimer  := time.NewTimer(stopDelay)
	eventTimer := time.NewTimer(eventDelay)
	for {
		select{
			case <-stopTimer.C:
				stopTimer.Stop()
				eventTimer.Stop()
				goto ExitSelect

			case <-eventTimer.C:
				eventTimer.Stop()
				if event != nil {
					event(t)
				}
		}
	}

ExitSelect:
	m1Peers := GetPeerManagerAvailablePeers(manager1)
	m2Peers := GetPeerManagerAvailablePeers(manager2)
	m3Peers := GetPeerManagerAvailablePeers(manager3)
	
	// Check all the peers discovered all the other peers
	if !MapOnlyContains(m1Peers, discovered1) {
		t.Errorf("Unexpected peers were discovered \n\treturned: %v \n\texpected: %v\n", m1Peers, discovered1)
		return
	}
	if !MapOnlyContains(m2Peers, discovered2) {
		t.Errorf("Unexpected peers were discovered \n\treturned: %v \n\texpected: %v\n", m2Peers, discovered2)
		return
	}
	if !MapOnlyContains(m3Peers, discovered3) {
		t.Errorf("Unexpected peers were discovered \n\treturned: %v \n\texpected: %v\n", m3Peers, discovered3)
		return
	}
	
	manager1.Stop()
	manager2.Stop()
	manager3.Stop()
}

// Test normal peer sesion
func TestPeerManager(t *testing.T) {

	seeds := []string {"127.0.0.1:6000",}
	manager1 := CreatePeerManager(6000, 6001, nil,  FullMode)
	manager2 := CreatePeerManager(7000, 7001, seeds, FullMode)
	manager3 := CreatePeerManager(8000, 8001, seeds, FullMode)

	allPeers := []string {"127.0.0.1:6001", "127.0.0.1:7001", "127.0.0.1:8001"}
	PeerManagerPropagationHelper(t, manager1, manager2, manager3, 
								allPeers, allPeers, allPeers, nil, time.Second, 2*time.Second)	
}

// Test the peer is deleted when it is unreachable
func TestPeerManagerDeletePeer(t *testing.T) {
	seeds := []string {"127.0.0.1:6000",}
	manager1 := CreatePeerManager(6000, 6001, nil,  FullMode)
	manager2 := CreatePeerManager(7000, 7001, seeds, FullMode)
	manager3 := CreatePeerManager(8000, 8001, seeds, FullMode)

	closePeer := func(t *testing.T) {
		manager1.Stop() // Just close http service
	}

	allPeers := []string {"127.0.0.1:7001", "127.0.0.1:8001"}
	noPeers  := []string {}
	PeerManagerPropagationHelper(t, manager1, manager2, manager3, 
		noPeers, allPeers, allPeers, closePeer, 2*time.Second, 4*time.Second)	

}

// Add unreachable marks until a peer address is marked, then rediscovered
func TestUnreachableMarks(t *testing.T) {
	seeds := []string {"127.0.0.1:6000",}
	manager1 := CreatePeerManager(6000, 6001, nil,  FullMode)
	manager2 := CreatePeerManager(7000, 7001, seeds, FullMode)
	manager3 := CreatePeerManager(8000, 8001, seeds, FullMode)

	// Change update period to lengthen the time between marking a peer unreachable 
	// and the next status update
	manager1.StatusUpdatePeriod=500*time.Millisecond
	manager2.StatusUpdatePeriod=500*time.Millisecond
	manager3.StatusUpdatePeriod=500*time.Millisecond

	markPeer := func(t *testing.T) {
		manager1.MarkPeerUnreachable("127.0.0.1:8001")
		manager1.MarkPeerUnreachable("127.0.0.1:8001")
		manager1.MarkPeerUnreachable("127.0.0.1:8001")
		available := GetPeerManagerAvailablePeers(manager1)
		expected  := []string {"127.0.0.1:6001", "127.0.0.1:7001"}
		if !MapOnlyContains(available, expected) {
			t.Errorf("Peer 127.0.0.1:8001 wasn't marked unreachable %v\n", available)
		}
	}

	// After some time has passed all the peers should be available again
	allPeers := []string {"127.0.0.1:6001", "127.0.0.1:7001", "127.0.0.1:8001"}
	PeerManagerPropagationHelper(t, manager1, manager2, manager3,
		allPeers, allPeers, allPeers, markPeer, 3200*time.Millisecond, 8*time.Second)
}

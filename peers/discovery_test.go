package peers

import (
	"testing"
	"time"
)



func CreatePeerHandler(peerPort, balancePort uint16, seeds []string, mode PeerMode) *PeerHandler {

	seedsCopy := make([]string, len(seeds))
	copy(seedsCopy, seeds)

	handler := &PeerHandler {
		Mode:         mode,
		PeerPort:     peerPort,
		BalancePort:  balancePort,
		Version:      "something something",
		SeedNodes:    seedsCopy,
		AllowLocalIP: true,
	
		AnnouncementPeriod:  time.Millisecond*50,
		PeerDiscoveryPeriod: time.Millisecond*50,
		StatusUpdatePeriod:  time.Millisecond*50,
		PeerMaxRetries:      3,
	}

	return handler
}

// MapContains checks map contains all the slice strings (and no more)
func MapOnlyContains(m map [string] bool,s []string) bool {

	for _, v := range s {
		if _, ok := m[v]; !ok {
			return false
		}
	}

	return len(m) == len(s)
}


// PeerPropagationHelper
func PeerPropagationHelper(t *testing.T, handler1, handler2, handler3 *PeerHandler, discovered []string, event func(), eventDelay time.Duration) {
	
	subPeer1 := handler1.Start()
	subPeer2 := handler2.Start()
	subPeer3 := handler3.Start()

	// Stop signal to stop peers
	stopTimer  := time.NewTimer(time.Second*5)
	eventTimer := time.NewTimer(eventDelay)
	discoveredPeer1 := make(map[string]bool)
	discoveredPeer2 := make(map[string]bool)
	discoveredPeer3 := make(map[string]bool)
	for {
		select{
			case msg := <- subPeer1:
				switch msg.typ {
				case AddPeerMsg:
					discoveredPeer1[msg.data.(string)] = true
				case DeletePeerMsg:
					delete(discoveredPeer1, msg.data.(string))
				}

			case msg := <- subPeer2:
				switch msg.typ {
				case AddPeerMsg:
					discoveredPeer2[msg.data.(string)] = true
				case DeletePeerMsg:
					delete(discoveredPeer2, msg.data.(string))
				}

			case msg := <- subPeer3:
				switch msg.typ {
				case AddPeerMsg:
					discoveredPeer3[msg.data.(string)] = true
				case DeletePeerMsg:
					delete(discoveredPeer3, msg.data.(string))
				}

			case <-stopTimer.C:
				stopTimer.Stop()
				handler1.Stop()
				handler2.Stop()
				handler3.Stop()
				goto ExitSelect

			case <-eventTimer.C:
				eventTimer.Stop()
				if event != nil {
					event()
				}
		}
	}

ExitSelect:

	// Check all the peers discovered all the other peers
	if !MapOnlyContains(discoveredPeer1, discovered) {
		t.Errorf("Not all peers were discovered \n\t%v \n\t%v\n", discoveredPeer1, discovered)
		return
	}
	if !MapOnlyContains(discoveredPeer2, discovered) {
		t.Errorf("Not all peers were discovered \n\t%v \n\t%v\n", discoveredPeer2, discovered)
		return
	}
	if !MapOnlyContains(discoveredPeer3, discovered) {
		t.Errorf("Not all peers were discovered \n\t%v \n\t%v\n", discoveredPeer3, discovered)
		return
	}
}


// TestPeerPropagation test nodes are propagated successfully
func TestPeerPropagation(t *testing.T) {
	
	seed := CreatePeerHandler(6000, 6001, nil, FullMode)
	seeds := []string{"localhost:6000"}
	peer1 := CreatePeerHandler(7000, 7001, seeds, FullMode)
	peer2 := CreatePeerHandler(8000, 8001, seeds, FullMode)
	
	// Check all the peers discovered all the other peers
	allPeers := [] string {"127.0.0.1:6001", "127.0.0.1:7001", "127.0.0.1:8001"}

	PeerPropagationHelper(t, seed, peer1, peer2, allPeers, nil, time.Second)
}

// TestFullNodes checks only fullmode peers are returned
func TestFullNodes(t *testing.T) {	
	seed := CreatePeerHandler(6000, 6001, nil, SeedMode)
	seeds := []string{"localhost:6000"}
	peer1 := CreatePeerHandler(7000, 7001, seeds, FullMode)
	peer2 := CreatePeerHandler(8000, 8001, seeds, FullMode)
	
	// Check all the peers discovered all the other peers
	allPeers := [] string {"127.0.0.1:7001", "127.0.0.1:8001"}
	
	PeerPropagationHelper(t, seed, peer1, peer2, allPeers, nil, time.Second)
}

// TestLocalIpFiltering check local ips are now allowed for remote peers
func TestLocalIpFiltering(t *testing.T) {	
	seed := CreatePeerHandler(6000, 6001, nil, SeedMode)
	seeds := []string{"localhost:6000"}
	peer1 := CreatePeerHandler(7000, 7001, seeds, FullMode)
	peer2 := CreatePeerHandler(8000, 8001, seeds, FullMode)
	
	seed.AllowLocalIP  = false
	peer1.AllowLocalIP = false
	peer2.AllowLocalIP = false
	
	// Check all the peers discovered all the other peers
	allPeers := [] string {}
	PeerPropagationHelper(t, seed, peer1, peer2, allPeers, nil, time.Second)
}

// TestUnreachableNotAdded in the first place
func TestUnreachableNotAdded(t *testing.T) {
	peer1 := CreatePeerHandler(6000, 6001, nil, SeedMode)
	seeds := []string{"localhost:6000", "localhost:9000", "unknown"} // With some fake ones
	peer2 := CreatePeerHandler(7000, 7001, seeds, LoadBalanceMode)
	peer3 := CreatePeerHandler(8000, 8001, seeds, FullMode)

	allPeers := [] string {"127.0.0.1:8001"}
	PeerPropagationHelper(t, peer1, peer2, peer3, allPeers, nil, time.Second)
}


// TestUnreachable peers are deleted
func TestUnreachablePeerDeleted(t *testing.T) {
	seed := CreatePeerHandler(6000, 6001, nil, SeedMode)
	seeds := []string{"localhost:6000"}
	peer1 := CreatePeerHandler(7000, 7001, seeds, FullMode)
	peer2 := CreatePeerHandler(8000, 8001, seeds, FullMode)
	
	closePeer := func() {
		peer1.shutdown() // Just close http service
	}

	// Check all the peers discovered all the other peers
	allPeers := [] string {"127.0.0.1:8001"}
	PeerPropagationHelper(t, seed, peer1, peer2, allPeers, closePeer, 2*time.Second)

}














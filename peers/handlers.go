package peers

import (
	"net/http"
	"encoding/json"
	"net"
	//"github.com/gorilla/mux"
)

type AnnouncementData struct {
	ip net.IP
	status Status
}


// Handle requests for the list of reachable peers
type PeerListHandler struct {
	commandCh chan *DiscoveryMsg
}

func NewPeerListHandler(commandCh chan *DiscoveryMsg) *PeerListHandler {
	h := PeerListHandler{commandCh: commandCh}
	return &h
}

func (p *PeerListHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// All active peers
	responseCh := make(chan []string)
	p.commandCh <- NewDiscoveryMsg(PeerListRequestMsg, responseCh)
	peers := <- responseCh

	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(writer).Encode(peers); err != nil {
		panic(err)
	}
}


// Handle requests for the status of this peer
type StatusHandler struct {
	commandCh chan *DiscoveryMsg
}

func NewStatusHandler(commandCh chan *DiscoveryMsg) *StatusHandler {
	h := StatusHandler{commandCh: commandCh}
	return &h
}

func (s *StatusHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	
	responseCh := make(chan Status)
	
	s.commandCh <- NewDiscoveryMsg(StatusRequestMsg, responseCh)
	status := <-responseCh

	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(writer).Encode(status); err != nil {
		panic(err)
	}
}


// Handle announcements from new peers
type PeerAnnouncementHandler struct {
	commandCh chan *DiscoveryMsg
}

func NewPeerAnnouncementHandler(commandCh chan *DiscoveryMsg) *PeerAnnouncementHandler {
	h := PeerAnnouncementHandler{commandCh: commandCh}
	return &h
}

func (p *PeerAnnouncementHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	var status Status

	err := json.NewDecoder(request.Body).Decode(&status)
	if err != nil {
		panic(err)
	}
	
	ip, _, err := ParseHost(request.RemoteAddr)
	if err != nil {
		panic(err)
	}

	data := AnnouncementData{ip: ip, status: status}
	p.commandCh <- NewDiscoveryMsg(PeerAnnouncementMsg, data)

	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	writer.WriteHeader(http.StatusOK)
}

package peers

import (
	"net/http"
	"github.com/gorilla/mux"
	"strings"
	"fmt"
	"github.com/secnot/gobalance/api"
)

const (
	// Get node peer list
	PeerListPath     = "peers"

	// Query node
	StatusPath		 = "status"

	// Announce
	AnnouncePath     = "announce"
)

type Route struct {
	Name        string  // Route name
	Method      string	// HTTP method GET, POST, PUT, ...
	Pattern     string	// Matching patters
	Handler     http.Handler
}



// Build route path by joining 
func BuildPath(urlPrefix string, path string) string {
    trimmedPref := strings.Trim(urlPrefix, "/")
    trimmedPath := strings.Trim(path, "/")

    if len(trimmedPref) == 0 {
        return fmt.Sprintf("/%v", trimmedPath)
    } else {
        return fmt.Sprintf("/%v/%v", trimmedPref, trimmedPath)
    }
}


func NewRouter(urlPrefix string, commandCh chan *DiscoveryMsg, log bool) *mux.Router {

	var routes = [...]Route {
		{
		StatusPath,
		"GET",
		StatusPath,
		NewStatusHandler(commandCh)},

		{
		PeerListPath,
		"GET",
		PeerListPath,
		NewPeerListHandler(commandCh)},

		{
		AnnouncePath,
		"POST",
		AnnouncePath,
		NewPeerAnnouncementHandler(commandCh)},
	}
	
	router := mux.NewRouter().StrictSlash(true)
    for _, route := range routes {
		handler :=  route.Handler
		if log {
        	handler = api.NewLoggerHandler(route.Handler, route.Name)
		}
        router.
            Methods(route.Method).
            Path(BuildPath(urlPrefix, route.Pattern)).
            Name(route.Name).
            Handler(handler)
    }
	return router
}

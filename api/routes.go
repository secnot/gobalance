package api

import (
	"fmt"
	"net/http"
	"strings"
	"github.com/gorilla/mux"
	"github.com/secnot/gobalance/logging"
)

const (
	// Get address balance
	BalancePath     = "address"

	// Get height
	HeightPath		 = "height"

	// Announce
	RecentTxPath     = "recent_tx"
)


type Route struct {
	Name        string  // Route name
	Method      string	// HTTP method GET, POST, PUT, ...
	Pattern     string	// Matching patters
	Handler     http.Handler
}

var routes = [...]Route {
	{
	BalancePath,
	"GET",
	"/address/{address}",
	http.HandlerFunc(BalanceHandlerFunc)},
	
	{
	HeightPath,
	"GET",
	"/height",
	http.HandlerFunc(HeightHandlerFunc)},

	{
	RecentTxPath,
	"GET",
	"/address/{address}/recent_tx",
	http.HandlerFunc(RecentTxHandlerFunc)},

	/*
	// Transactions involving this address in the last few blocks
	{"recent_transactions",
	"GET",
	"/address/{address}/recent_transactions",
	http.HandlerFunc(recentTransactions)},

	// Height and date for the top block
	{"height",
	"GET",
	"/height",
	http.HandlerFunc(heightHandler)},
	*/
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

func NewRouter(urlPrefix string) *mux.Router {

    router := mux.NewRouter().StrictSlash(true)
    for _, route := range routes {
		loggedHandler := logging.NewLoggerHandler(route.Handler, route.Name)
        router.
            Methods(route.Method).
            Path(BuildPath(urlPrefix, route.Pattern)).
            Name(route.Name).
            Handler(loggedHandler)
    }

    return router
}

package api

import (
	"fmt"
	"net/http"
	"strings"
	"github.com/gorilla/mux"
	"github.com/secnot/gobalance/logging"
	"github.com/secnot/gobalance/api/common"
	
	"github.com/secnot/gobalance/height"
	"github.com/secnot/gobalance/balance"
	"github.com/secnot/gobalance/recent_tx"
)

const (
	// Get address balance
	BalancePath     = "address"

	// Get height
	HeightPath		 = "height"

	// Announce
	RecentTxPath     = "recent_tx"
)

type HandlerFuncConstructor func (*balance.BalanceCache, *recent_tx.RecentTxCache, *height.HeightCache) http.Handler

type Route struct {
	Name        string  // Route name
	Method      string	// HTTP method GET, POST, PUT, ...
	Pattern     string	// Matching patters
	HandlerConstructor HandlerFuncConstructor
}

var routes = [...]Route {
	{
	api_common.BalancePath,
	"GET",
	"/address/{address}",
	BalanceHandlerConstructor},
	
	{
	api_common.HeightPath,
	"GET",
	"/height",
	HeightHandlerConstructor},

	{
	api_common.RecentTxPath,
	"GET",
	"/address/{address}/recent_tx",
	RecentTxHandlerConstructor},

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

// NewRouter
func NewRouter(urlPrefix string,
	balanceC  *balance.BalanceCache, 
	recentTxC *recent_tx.RecentTxCache,
	heightC   *height.HeightCache) *mux.Router {

    router := mux.NewRouter().StrictSlash(true)
   
	// Add routes
	for _, route := range routes {
		handler := route.HandlerConstructor(balanceC, recentTxC, heightC)
		loggedHandler := logging.NewLoggerHandler(handler, route.Name)
        router.
            Methods(route.Method).
            Path(BuildPath(urlPrefix, route.Pattern)).
            Name(route.Name).
            Handler(loggedHandler)
    }

    return router
}

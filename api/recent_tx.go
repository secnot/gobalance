package api

import (
	"net/http"
	"encoding/json"

	"github.com/secnot/gobalance/recent_tx"
	"github.com/gorilla/mux"
)

// Recent transactions Handler function
func RecentTxHandlerFunc(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	address := vars["address"]
	
	// Get balance from crawler
	//transactions, blocks := recent.GetRecentTx(address)
	transactions, _ := recent.GetRecentTx(address)

	response := make([]Tx, len(transactions))
	for n, tx := range transactions {
	
		response[n] = Tx {
			Hash: tx.Hash.String(),
			
		}	
	}

	/* TODO: Handle errors and response error message
	if balance.Err != nil {
		...
	}
	*/
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		panic(err)
	}
}



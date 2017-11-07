package api

import (
	"net/http"
	"encoding/json"

	"github.com/secnot/gobalance/balance"
	"github.com/gorilla/mux"
)



// Balance Handler function
func BalanceHandlerFunc(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	address := vars["address"]
	
	// Get balance from crawler
	bal := balance.GetBalance(address)
	
	/* TODO: Handle errors and response error message
	if balance.Err != nil {
		...
	}
	*/
	response := Address {
		Address: address,
		Balance: bal,
	}
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		panic(err)
	}
}



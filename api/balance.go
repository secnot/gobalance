package api

import (
	"net/http"
	"encoding/json"

	"github.com/secnot/gobalance/api/common"
	"github.com/secnot/gobalance/interfaces"
	"github.com/secnot/gobalance/utils"
	"github.com/gorilla/mux"
)



// Balance Handler function
func BalanceHandlerConstructor(balanceC interfaces.BalanceCache, recentC interfaces.RecentTxCache, heightC interfaces.HeightCache) http.Handler {

	handler := func (writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		address := vars["address"]
			
		// Use requester ip	as its identification
		ip, _, err := utils.ParseHost(request.RemoteAddr)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		// Request balance
		bal, err := balanceC.GetBalance(address, ip)
		if err != nil {
			 http.Error(writer, err.Error(), http.StatusInternalServerError)
			 return
		}
		
		// Send response back.
		response := api_common.Address {
			Address: address,
			Balance: bal,
		}
		writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			panic(err)
		}

	}	
	
	return http.HandlerFunc(handler)
}


package api

import (
	"net/http"
	"encoding/json"

	"github.com/secnot/gobalance/api/common"
	"github.com/secnot/gobalance/height"
	"github.com/secnot/gobalance/balance"
	"github.com/secnot/gobalance/recent_tx"
	"github.com/gorilla/mux"
)

func RecentTxHandlerConstructor(balanceC *balance.BalanceCache, recentC *recent_tx.RecentTxCache, heightC *height.HeightCache) http.Handler {
	
	handler := func (writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		address := vars["address"]
		
		// Get balance from crawler
		transactions, _, err := recentC.GetRecentTx(address)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		response := make([]api_common.Tx, len(transactions))
		for n, tx := range transactions {
		
			response[n] = api_common.Tx {
				Hash: tx.Hash.String(),
				
			}	
		}

		writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			panic(err)
		}
	}

	return http.HandlerFunc(handler)
}	


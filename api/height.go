package api

import (
	"net/http"
	"encoding/json"

	"github.com/secnot/gobalance/height"
	"github.com/secnot/gobalance/balance"
	"github.com/secnot/gobalance/recent_tx"
)

func HeightHandlerConstructor(balanceC *balance.BalanceCache, recentC *recent_tx.RecentTxCache, heightC *height.HeightCache) http.Handler {
	handler := func(writer http.ResponseWriter, request *http.Request) {
		response := heightC.GetHeight()
		writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			panic(err)
		}
	}

	return http.HandlerFunc(handler)
}


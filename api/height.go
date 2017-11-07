package api

import (
	"net/http"
	"encoding/json"

	"github.com/secnot/gobalance/height"
)


// Balance Handler function
func HeightHandlerFunc(writer http.ResponseWriter, request *http.Request) {
	response := height.GetHeight()
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		panic(err)
	}
}



package api

import (
	"log"
	"net/http"
)

func StartApi(address string, urlPrefix string) {
	router := NewRouter(urlPrefix)
	log.Fatal(http.ListenAndServe(":8080", router))
}

/*
logger implements a simple http.Handler used to log requests.
*/
package logging

import (
    "log"
    "net/http"
    "time"
)

type LoggingHandler struct {
	next http.Handler
	name string
}

func NewLoggerHandler(h http.Handler, name string) http.Handler {
	return LoggingHandler{
		next: h,
		name: name,		
	}
}

func (h LoggingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	h.next.ServeHTTP(rw, r)
	log.Printf(
			"[%s] - (%s) %s (%s)",
			h.name,
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
}

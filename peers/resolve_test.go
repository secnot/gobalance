package peers

import (
	"testing"
)


func TestResolveHostnameRoutine(t *testing.T) {

	requestCh  := make(chan string, 5)
	responseCh := make(chan string, 5)
	exitCh     := make(chan bool)

	go resolveHostnameRoutine(requestCh, responseCh, exitCh)

	// send something to resolve
	requestCh <- "localhost:8000"
	resolved := <- responseCh

	if resolved != "127.0.0.1:8000"{
		t.Error("Expecting '127.0.0.1:8000' returned %v", resolved)
		return
	}

	exitCh <- true
}

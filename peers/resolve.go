package peers

import (
	"time"
	"log"
	"github.com/secnot/gobalance/utils"
)

const(
	// Max
	lookupMaxRetries = 5
	
	//
	lookupRetryPeriod = 500*time.Millisecond
)

// getNextHostname returns one of the hostname with the least retries
func getNextHostname(hosts map[string]int) (next string, ok bool) {

	current_retries := lookupMaxRetries * 20
	current_host    := ""
	ok = false
	
	for host, retries := range hosts {
		if retries < current_retries {
			current_host    = host
			current_retries = retries
			ok = true
		}

		if retries == 0 {
			break
		}
	}

	return current_host, ok
}

// doResolve
func doResolve(hosts map[string]int, resolvedCh chan string) {		
	
	hostname, ok := getNextHostname(hosts)
	if !ok {
		return
	}

	result, err := utils.ResolveHostname(hostname)
	if err != nil {
		// Log on first error
		if hosts[hostname] == 0 {
			log.Printf("Resolving %v: %v", hostname, err)
		}

		// Update retries
		hosts[hostname] = hosts[hostname] + 1
		if hosts[hostname] > lookupMaxRetries {
			delete(hosts, hostname)
		}
		return
	}

	// Return resolved hostname
	delete(hosts, hostname)
	resolvedCh <- result
}

// resolvePeerHostnameRoutine receives peer hostnames and returns resolved addresses.
func resolveHostnameRoutine(requestCh, resolvedCh chan string, exitCh chan bool) {

	// TODO: Use a priority queue if there are too many hostnames
	hosts := make(map[string]int)
	ticker := time.NewTicker(lookupRetryPeriod)
	
	for {

		select {
		case hostname := <-requestCh:
			// Check the address needs to be resolved
			ip, port, err := utils.ParseHost(hostname)
			if err != nil {
				hosts[hostname] = 0
			} else {
				resolvedCh <- utils.HostToString(ip, port)
			}

		case <-exitCh:
			ticker.Stop()
			return

		case <-ticker.C:
			doResolve(hosts, resolvedCh)
		}
	}
}

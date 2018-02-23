package peers

import (
	"testing"
	"net"
)

func TestParseHost(t *testing.T) {

	ip, port, err := ParseHost("192.167.1.1:88")
	if err != nil {
		t.Error(err)
		return
	}
	if !ip.Equal(net.IPv4(192, 167, 1, 1)) || port != 88 {
		t.Error("ParseHost(): Unexpected ", ip, port)
		return
	}

	// no ip
	_, _, err = ParseHost("hostname:6666")
	if err == nil {
		t.Error("Expecting an error")
		return
	}

	// Only ip
	_, _, err = ParseHost("127.0.0.1")
	if err == nil {
		t.Error("Expecting an error")
		return
	}

	// ip port with a path
	_, _, err = ParseHost("88.77.0.2:122/asdfasf")
	if err == nil {
		t.Error("Expecting an error")
		return
	}

	// empty
	_, _, err = ParseHost("")
	if err == nil {
		t.Error("Expecting an error")
		return
	}

	// only port
	_, _, err = ParseHost(":55")
	if err == nil {
		t.Error("Expecting an error")
		return
	}	
	
	_, _, err = ParseHost(" :55")
	if err == nil {
		t.Error("Expecting an error")
		return
	}
}


func TestResolveHostname(t *testing.T) {

	// Test valid hostname
	resolved, err := ResolveHostname("localhost:8080")
	if err !=  nil {
		t.Error(err)
		return
	}

	if resolved != "127.0.0.1:8080" && resolved != "::1:8080"{
		t.Errorf("Resolved wrong ip%v", resolved)
		return
	}

	// Test hostname without port
	resolved, err = ResolveHostname("localhost")
	if err == nil {
		t.Error("Expection an error while resolving hostname without port")
		return
	}

	// Test ipv4:port
	resolved, err = ResolveHostname("80.33.12.12:8080")
	if err !=  nil {
		t.Error(err)
		return
	}

	if resolved != "80.33.12.12:8080"{
		t.Errorf("Expecting 80.33.12.12:8080 returned %v", resolved)
		return
	}

	// Test ipv4 without port	
	resolved, err = ResolveHostname("80.33.12.12")
	if err ==  nil {
		t.Error("Expecting an error while resolving ipv4 without port")
		return
	}

	// Test ipv6:port	
	hostname := "[2001:0db8:0000:0000:0000:ff00:0042:8329]:8080"
	resolved, err = ResolveHostname(hostname)
	if err !=  nil {
		t.Error(err)
		return
	}

	ip1, port1, err := ParseHost(hostname)
	ip2, port2, err := ParseHost(resolved)
	if !ip1.Equal(ip2) {
		t.Errorf("the resolved ipv6 is not the same %v != %v", ip1, ip2)
		return
	}
	if port1 != port2 {
		t.Errorf("the resolved port is not the same %v != %v", port1, port2)
		return
	}

	// Test ipv6 without port
	hostname = "[2001:0db8:0000:0000:0000:ff00:0042:8329]"
	resolved, err = ResolveHostname(hostname)
	if err ==  nil {
		t.Error("Expection an error while resolving ipv6 without port")
		return
	}
}


func TestGetLocalIPs(t *testing.T) {

	// Local ips including loopback 
	loopbackFound := false
	for _, ip := range GetLocalIPs(true) {
		if ip.IsLoopback() {
			loopbackFound = true
			break
		}
	}
	if !loopbackFound {
		t.Error("No loopback address found")
		return
	}

	// Local ips not including loopback
	for _, ip := range GetLocalIPs(false) {
		if ip.IsLoopback() {
			t.Error("Loopback address found")
			return
		}
	}
}

func TestGetOutboundIP(t *testing.T) {
	ip, err := GetOutboundIP()
	if err != nil {
		t.Error(err)
		return
	}
	if ip.IsLoopback() {
		t.Error("Returned loopback address %v", ip)
	}
}

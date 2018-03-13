package utils

import (
	"net"
	"fmt"
	"math"
	"strconv"
	"errors"
)

var ErrInvalidIpAddress  = errors.New("Invalid ip address")
var ErrInvalidPortNumber = errors.New("Invalid port number") 

// isIP4 return true if the ip is ipv4
func isIP4(ip net.IP) bool {
	return ip.To4() != nil
}

// HostToString constructs a host string from ip and port
func HostToString(ip net.IP, port uint16) string {	
	if isIP4(ip) {
		return fmt.Sprintf("%v:%v", ip.String(), port)
	} else {
		return fmt.Sprintf("[%v]:%v", ip.String(), port)
	}
}

// ParseHost parses "ip:port" string
func ParseHost(host string) (ip net.IP, port uint16, err error) {	
	hostStr, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return net.IPv4zero, 0, err
	}

	ip = net.ParseIP(hostStr)
	if ip == nil {
		return net.IPv4zero, 0, ErrInvalidIpAddress
	}

	p, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || p < 1 || p > math.MaxUint16 {
		return net.IPv4zero, 0, ErrInvalidPortNumber
	}

	return ip, uint16(p), nil
}

// ResolveHostname tries to resolve ip address from a peer hostname:port string
func ResolveHostname(hostname string) (resolved string, err error) {	

	//
	host, aport, err := net.SplitHostPort(hostname)
	if err != nil {
		return "", err
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return "", err
	}

	// Select the first ipv4 when there is a choice
	selection := ips[0]
	for _, ip := range ips {

		if isIP4(ip) {
			selection = ip
			break
		}
	}

	port, err := strconv.ParseUint(aport, 10, 16)
	if err != nil {
		return "", err
	}

	return HostToString(selection, uint16(port)), nil
}


// GetLocalIPs returns the list of all non loopback or not local IPs
func GetLocalIPs(includeLoopback bool) []net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if !includeLoopback && ipnet.IP.IsLoopback() {
				continue
			}
			ips = append(ips, ipnet.IP)
		}
	}

	return ips
}

// GetOUtboundIP returns the preferred outbound ip
func GetOutboundIP() (net.IP, error) {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return net.IPv4zero, err
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP, nil
}


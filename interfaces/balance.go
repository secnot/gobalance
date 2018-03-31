package interfaces

import (
	"net" 
)

type BalanceCache interface {

	// Get address balance
	GetBalance(address string, ip net.IP) (balance int64, err error)
	
	// Stop cache
	Stop()
}

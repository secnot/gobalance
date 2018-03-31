package interfaces

import (
	"github.com/secnot/gobalance/primitives"
)

type RecentTxCache interface {
	GetRecentTx(address string) ([]*primitives.Tx, []*primitives.Block, error)
	Stop()
}

package balance


import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)



type TxOut struct {
	TxHash chainhash.Hash
	Nout uint32
	Addr *string
	Value int64
}

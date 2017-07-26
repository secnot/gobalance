//TxRecord is a ss
package crawler


import (
	"fmt"
	"github.com/btcsuite/btcd/wire"
	"github.com/secnot/gobalance/primitives"
)

// 
type TxRecord struct {
	Outputs []*TxRecordOut
	unspent uint32// number of unspent outputs remainnig in the record
}

// 
type TxRecordOut struct {
	Addr string
	Value int64
}

// NewTxRecord constructs  a TxRecord from raw wire transaction
func NewTxRecord(tx *wire.MsgTx) *TxRecord {
	
	outputs := make([]*TxRecordOut, 0, len(tx.TxOut))
	unspent := uint32(0)
	for _, txout := range tx.TxOut {
		outputs = append(outputs, NewTxRecordOut(txout))
		
		// Only consider spendable outputs
		if txout.Value != 0 {
			unspent++
		}
	}

	return &TxRecord{
		Outputs: outputs,
		unspent: unspent,
	}
}


// NewTxRecordOut cosntructs a TxRecordOut from a raw transaction output
func NewTxRecordOut(txout *wire.TxOut) *TxRecordOut {

	return &TxRecordOut{
		Addr: primitives.PkScriptToAddr(txout.PkScript),
		Value: txout.Value,
	}
}


func (t *TxRecordOut) String() string {
	return fmt.Sprintf("TxRecordOut(%v, %v)", t.Addr, t.Value)
}

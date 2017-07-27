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


var TxCount int64
var TxOutCount int64


// NewTxRecord constructs  a TxRecord from raw wire transaction
func NewTxRecord(tx *wire.MsgTx) *TxRecord {
	
	outputs := make([]*TxRecordOut, len(tx.TxOut))
	for n, txout := range tx.TxOut {
		outputs[n] = NewTxRecordOut(txout)
		TxOutCount++
	}

	TxCount++
	return &TxRecord{
		Outputs: outputs,
		unspent: uint32(len(tx.TxOut)),
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

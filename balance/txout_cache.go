package balance


import (
	_ "fmt"
	"github.com/btcsuite/btcd/wire"
	_ "github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/secnot/simplelru"
)



type TxOutKey struct {
	TxHash chainhash.Hash
	Nout uint32
}

type TxOutData struct {
	Addr *string
	Value int64
}


// Unspent transaction output pool
type TxOutPool struct {
	pool *simplelru.LRUCache
}

func NewTxOutPool() (*TxOutPool) {
	return &TxOutPool{
		pool: simplelru.NewLRUCache(2000000, 1000),
	}
}


// Remove spent TxOut from pool
func (t *TxOutPool) DelTxOut(txHash chainhash.Hash, nOut uint32) {
	t.pool.Remove(TxOutKey{TxHash: txHash, Nout: nOut})
}

// Add TxOut to pool
func (t *TxOutPool) SetTxOut(txHash chainhash.Hash, nOut uint32, txOut *TxOut) {
	txOutData := TxOutData{Addr: txOut.Addr, Value: txOut.Value}
	t.pool.Set(TxOutKey{TxHash: txHash, Nout: nOut}, txOutData)
}


// Get value for unspent Tx Output
func (t *TxOutPool) GetTxOut(txHash chainhash.Hash, nOut uint32) (txOut *TxOut, ok bool) {
	data, ok := t.pool.Get(TxOutKey{TxHash: txHash, Nout: nOut})
	if !ok {
		txOut, ok = nil, false
	} else {
		txoutData := data.(TxOutData)
		txOut = &TxOut{
			TxHash: txHash,
			Nout: nOut,
			Addr: txoutData.Addr,
			Value: txoutData.Value,
		}
		ok = true
	}	
	return
}



// Add transaction outputs and remove transactions inputs to and from the pool
func (t *TxOutPool) AddTx(tx *wire.MsgTx) {
	txHash := tx.TxHash()
	
	// First add new outputs
	for N, vout := range tx.TxOut {
		//TODO: Decode Vout output address if any
		//  pkScript := vout.PkScript
		//	scriptClass, addressLst, reqSigs, err := txscript.ExtractPkScriptAddrs(pkScript, chainParams????)
		txOut := TxOut{ 
			TxHash: txHash,
			Nout: uint32(N),
			Value: vout.Value,
			Addr: nil,
		}
		t.SetTxOut(txHash, uint32(N), &txOut)
	}

	// Delete used inputs
	// TODO: Not required if using a lru .....
	//for _, vin := range tx.TxIn {
	//	// TODO: Ignore Coinbase VIN
	//	t.DelTxOut(vin.PreviousOutPoint.Hash, vin.PreviousOutPoint.Index)
	//}
}


// Update pool with block transactions
func (t *TxOutPool) AddBlock(block *wire.MsgBlock) {
	for _, tx := range block.Transactions {
		t.AddTx(tx)
	}
}


func BlockFactory()

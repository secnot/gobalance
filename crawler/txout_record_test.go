package crawler


import (
	"testing"
	"github.com/btcsuite/btcd/wire"

	"github.com/btcsuite/btcutil"
	"github.com/secnot/gobalance/primitives"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)


func newAddressPubKey(serializedPubKey []byte) btcutil.Address {
	addr, err := btcutil.NewAddressPubKey(serializedPubKey, primitives.DefaultChainParams)
	if err != nil {
		panic("invalid public key in test source")
	}

	return addr
}


//
func TestTxRecord(t *testing.T) {

	hashStr := "0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098"
	hash, _ := chainhash.NewHashFromStr(hashStr)
	tx := wire.NewMsgTx(1)
	
	record := NewTxRecord(hash, tx)
	if record == nil {
		t.Error("Error creating TxRecord")
	}
}

// Basic TxRecordOut creations errors
func TestTxRecordOut(t *testing.T) {

	t.Parallel()
	
	// Test building TxRecordOut from TxRecord
	txValue := int64(5000000000)

	// tandard p2pk with uncompressed pubkey
	script := primitives.HexToBytes(
		"410411db93e1dcdb8a016b49840f8c53bc1eb68a382e" +
		"97b1482ecad7b148a6909a5cb2e0eaddfb84ccf97444" +
		"64f82e160bfa9b8b64f9d4c03f999b8643f656b412a3ac")

	addr := newAddressPubKey(primitives.HexToBytes(
			"0411db93e1dcdb8a" +
			"016b49840f8c53bc1eb68a382e97b1482eca" +
			"d7b148a6909a5cb2e0eaddfb84ccf9744464" +
			"f82e160bfa9b8b64f9d4c03f999b8643f656" +
			"b412a3"))

	txOut := wire.NewTxOut(txValue, script)
	
	txRecordOut := NewTxRecordOut(txOut)
	if txRecordOut.Addr != addr.EncodeAddress() {
		t.Errorf("Unexpected address %v", txRecordOut.Addr)
	}
	if txRecordOut.Value != txValue {
		t.Errorf("Unexpected value %v", txRecordOut.Value)
	}
}

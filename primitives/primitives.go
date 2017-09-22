package primitives


import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

const (
	// defaultTransactionAlloc is the default size used for the backing array
	// for transactions.  The transaction array will dynamically grow as needed, but
	// this figure is intended to provide enough space for the number of
	// transactions in the vast majority of blocks without needing to grow the
	// backing array multiple times.
	defaultTransactionAlloc = 2048

	// defaultTxInOutAlloc is the default size used for the backing array for
	// transaction inputs and outputs.  The array will dynamically grow as needed,
	// but this figure is intended to provide enough space for the number of
	// inputs and outputs in a typical transaction without needing to grow the
	// backing array multiple times.
	defaultTxInOutAlloc = 15
)

// Hash for the first block in the blockchain
var MainNetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
		0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
})

// Empty hash 
var ZeroHash = chainhash.Hash([chainhash.HashSize]byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
})


// Operate on the TestNet Bitcoin network
var DefaultChainParams = &chaincfg.MainNetParams

// Select chain operationOperate on MainNet network
// &chaincfg.MainNetParams
// &chaincfg.TestNet3Params
func SelectChain(chain *chaincfg.Params) {
	DefaultChainParams = chain 
}



type Block struct {
	Hash         chainhash.Hash 
	PrevHash     chainhash.Hash
	Height       uint64

	Transactions []*Tx
}

type Tx struct {
	Hash *chainhash.Hash // Transaction hash
	In   []*TxOut        // Inputs substituted by the TxOut they point to
	Out  []*TxOut        // Outputs
}

type TxOut struct {
	TxHash *chainhash.Hash // Hash for the transaction containing the TxOut
	Nout   uint32          // Output number
	Addr   string          // Bitcoin address from pkScript
	Value  int64           // Output ammount
}



func NewBlock(hash chainhash.Hash, prev chainhash.Hash, height uint64) *Block {
	return &Block{
		Hash:         hash,
		PrevHash:     prev,
		Height:       height,
	}
}

func (b *Block) AddTx(tx *Tx) {
	if b.Transactions == nil {
		b.Transactions = make([]*Tx, 0, defaultTransactionAlloc)
	}
	b.Transactions = append(b.Transactions, tx)
}


func NewTx(hash *chainhash.Hash) *Tx {
	return &Tx{
		Hash: hash,
		In:   make([]*TxOut, 0, defaultTxInOutAlloc),
		Out:  make([]*TxOut, 0, defaultTxInOutAlloc),
	}
}

// Add Input to transaction
func (t *Tx) AddIn(in *TxOut) {
	t.In = append(t.In, in)
}

// Add Output to transaction
func (t *Tx) AddOut(out *TxOut) {
	t.Out = append(t.Out, out)
}


func NewTxOut(txHash *chainhash.Hash, nout uint32, address string, value int64) *TxOut {
	return &TxOut{
		TxHash: txHash,
		Nout:   nout,
		Addr:   address,
		Value:  value,
	}
}

// PkScriptToAddr extracts the bitcoin address from a wire.TxOut.PkScript
func PkScriptToAddr(pkScript []byte) string {
	// See http://godoc.org/github.com/btcsuite/btcd/txscript#example-ExtractPkScriptAddrs
	scriptClass, addresses, _, err := txscript.ExtractPkScriptAddrs(pkScript, 
		DefaultChainParams)
	if err != nil || len(addresses) == 0 {
		return ""
	}

	switch scriptClass {
	case txscript.PubKeyHashTy:
		return addresses[0].EncodeAddress()
	case txscript.PubKeyTy:
		return addresses[0].EncodeAddress()
	case txscript.ScriptHashTy:
		return addresses[0].EncodeAddress()
	// Remaining cases to default no address
	// case txscript.NonStandardTy:
	// case txscript.MultiSigTy:
	// case txscript.NullDataTy:
	default:
		return ""
	}
}

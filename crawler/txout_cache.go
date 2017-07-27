package crawler


import (
	"log"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	//"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcrpcclient"
	"github.com/secnot/gobalance/primitives"
)


const (
	// Number of transactions cached by default
	defaultCacheSize = 2000000

	//
	defaultCachePruneSize = 1000

	// Retry period between failed requests
	rpcRetryPeriod = 5000

	// Max number of retries 
	rpcRetryNumber = 10

)


// Transaction exists but it doesn't contain requested TxOut
type ErrMissingTxOut struct {
	message string
}

func NewErrMissingTxOut(message string) *ErrMissingTxOut {
	return &ErrMissingTxOut{
		message: message,
	}
}

func (e *ErrMissingTxOut) Error() string {
	return e.message
}

// Unable to obtain the transaction 
type ErrRPCFail struct{
	message string
}

func NewErrRPCFail(message string) *ErrRPCFail {
	return &ErrRPCFail{
		message: message,
	}
}

func (e *ErrRPCFail) Error() string {
	return e.message
}


// Unable create TxRecord from transaction
type ErrInvalidTx struct {
	message string
}

func NewErrInvalidTx(message string) *ErrInvalidTx {
	return &ErrInvalidTx{
		message: message,
	}
}

func (e *ErrInvalidTx) Error() string {
	return e.message
}



// Unspent transaction output pool
type TxOutCache struct {
	//TODO: Substitute simplelru with a more efficient storage method
	cache map[chainhash.Hash]TxRecord
	rpc  *btcrpcclient.Client
}


// NewTxOutCache allocates a new empty cache
func NewTxOutCache(client *btcrpcclient.Client) (*TxOutCache) {
	
	return &TxOutCache{
		cache: make(map[chainhash.Hash]TxRecord),
		rpc:  client,
	}
}


func (t *TxOutCache) getTxOut(txHash *chainhash.Hash, nOut uint32, peek bool) *primitives.TxOut {
	record, ok := t.cache[*txHash]
	if !ok {
		return nil
	}

	if nOut+1 > uint32(len(record.Outputs)) {
		log.Panicf("Transaction %v doesn't have output %v", *txHash, nOut)
	}

	out := record.Outputs[nOut]
	if out == nil {
		// The output was already used, or didn't contain relevant information
		return nil
	}

	// If it isn't a peek delete the txout and also the transaction it's empty
	if !peek {
		record.Outputs[nOut] = nil
		record.unspent--
		if record.unspent == 0 {
			delete(t.cache, *txHash)
		}
	}

	return primitives.NewTxOut(txHash, nOut, out.Addr, out.Value)
}

// Get value for unspent Tx Output
func (t *TxOutCache) GetTxOut(txHash *chainhash.Hash, nOut uint32) *primitives.TxOut {
	return t.getTxOut(txHash, nOut, false)
}

// PeekTxOut returns cached txout but without deleting the used TxOut from cache
func (t *TxOutCache) PeekTxOut(txHash *chainhash.Hash, nOut uint32) *primitives.TxOut {
	return t.getTxOut(txHash, nOut, true)
}

// Add transaction outputs and remove transactions inputs to and from the pool
func (t *TxOutCache) SetTx(txHash *chainhash.Hash, tx *wire.MsgTx) {
	record := NewTxRecord(tx)

	// Eliminate outputs without an address ot with 0 value because they will not
	// affect balance calculations
	for n, txOut := range record.Outputs {
		if txOut.Value == 0 || txOut.Addr == "" {
			record.Outputs[n] = nil
			record.unspent--
		}
	}

	// It it has unspent outputs add transaction to cache
	if record.unspent > 0 {
		t.cache[*txHash] = *record
	}
}

// Remove transaction record from cache if it's still cached
func (t *TxOutCache) DelTx(txHash *chainhash.Hash) {
	delete(t.cache, *txHash)
}


func (t *TxOutCache) Len() int {
	return len(t.cache)
}

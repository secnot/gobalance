package crawler


import (
	"fmt"
	"time"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	//"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcrpcclient"
	"github.com/secnot/simplelru"
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
	cache *simplelru.LRUCache
	rpc  *btcrpcclient.Client
}

//
func newRPCTxFetchFunc(client *btcrpcclient.Client) simplelru.FetchFunc {

	fetchFunc := func(key interface{}) (value interface{}, ok bool) {
		retryCount := 0 // Number of retries used for the current
		RETRY:
		for {
			hash := key.(chainhash.Hash)
			tx, err := client.GetRawTransaction(&hash)
			if err != nil {
				if jerr, ok := err.(*btcjson.RPCError); ok {
					switch jerr.Code {
					case btcjson.ErrRPCClientInInitialDownload:
					case btcjson.ErrRPCClientNotConnected:
						time.Sleep(rpcRetryPeriod*time.Millisecond)
						retryCount++
						break RETRY
					default:
						return nil, false
					}
				}		
			}

			// Return decoded transaction record
			return NewTxRecord(tx.MsgTx()), true
		}	
		return nil, false
	}

	return fetchFunc
}

// NewTxOutCache allocates a new empty cache
func NewTxOutCache(client *btcrpcclient.Client) (*TxOutCache) {
	

	fetchFunc      := newRPCTxFetchFunc(client)
	fetchWorkers   := uint32(1)
	fetchQueueSize := fetchWorkers*2+1

	cache := simplelru.NewFetchingLRUCache(
		defaultCacheSize, defaultCachePruneSize,
		fetchFunc, fetchWorkers, fetchQueueSize)

	return &TxOutCache{
		cache: cache,
		rpc:  client,
	}
}

// Resize cache and set new prune size
func (t *TxOutCache) Resize(size int, prune int) {
	t.cache.Resize(size, prune)
}

func (t *TxOutCache) getTxOut(txHash *chainhash.Hash, nOut uint32, peek bool) (*primitives.TxOut, error) {
	txRecordInterface, ok := t.cache.Get(*txHash)
	if !ok {
		errMsg := fmt.Sprintf("Unable to retrieve Tx(%v)", txHash)
		return nil, NewErrRPCFail(errMsg)
	}

	record := txRecordInterface.(*TxRecord)
	if nOut+1 > uint32(len(record.Outputs)) {
		errMsg := fmt.Sprintf("Tx() doesn't containts %v", txHash, nOut)
		return nil, NewErrMissingTxOut(errMsg)
	}

	out := record.Outputs[nOut]
	if out == nil {
		// Each TxOut is only retrieved once and then deleted, so this shouldn't
		// happen, but just in case discard the record and retrieve the full 
		// transaction again.
		t.cache.Remove(*txHash)
		return t.getTxOut(txHash, nOut, peek)
	}

	// If it isn't a peek delete the txout and also the transaction it's empty
	if !peek {
		record.Outputs[nOut] = nil
		record.unspent--
		if record.unspent == 0 {
			t.cache.Remove(*txHash)
		}
	}

	return primitives.NewTxOut(txHash, nOut, out.Addr, out.Value), nil
}

// Get value for unspent Tx Output
func (t *TxOutCache) GetTxOut(txHash *chainhash.Hash, nOut uint32) (*primitives.TxOut, error) {
	return t.getTxOut(txHash, nOut, false)
}

// PeekTxOut returns cached txout but without deleting the used TxOut from cache
func (t *TxOutCache) PeekTxOut(txHash *chainhash.Hash, nOut uint32) (*primitives.TxOut, error) {
	return t.getTxOut(txHash, nOut, true)
}

// Add transaction outputs and remove transactions inputs to and from the pool
func (t *TxOutCache) SetTx(txHash *chainhash.Hash, tx *wire.MsgTx) error {
	t.cache.Set(*txHash, NewTxRecord(tx))
	return nil
}

// Remove transaction record from cache if it's still cached
func (t *TxOutCache) DelTx(txHash *chainhash.Hash) {
	t.cache.Remove(*txHash)
}


// Update pool with block transactions
func (t *TxOutCache) AddBlock(block *wire.MsgBlock) error {
	for _, tx := range block.Transactions {
		hash := tx.TxHash()
		err := t.SetTx(&hash, tx)
		if err != nil {
			return err
		}
	}
	return nil
}



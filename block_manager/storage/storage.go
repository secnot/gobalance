package storage

import (
	"errors"
	"github.com/secnot/gobalance/primitives"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)


type TxOutId struct {
	// Transaction hash
	TxHash chainhash.Hash // Transaction hash

	// Transaction output number 
	Nout uint32
}

func NewTxOutId(hash *chainhash.Hash, nout uint32) *TxOutId {
	return &TxOutId{
		TxHash: *hash,
		Nout:   nout,
	}
}

type TxOutData struct {
	// Output value
	Value   int64

	// Redeem address for the transaction output
	Addr string
}

func NewTxOutData(address string, value int64) *TxOutData {
	return &TxOutData{
		Addr:  address,
		Value: value,
	}
}


// Memory and SQL storage interface:
type Storage interface {

	// Number of Utxo stored
	Len() (length int, err error)
	
	// Get last block height and hash (return -1, emptyHash if not stored)
	GetLastBlock() (height int64, hash chainhash.Hash, err error)
	
	// Set last commited block height and hash
	SetLastBlock(height int64, hash chainhash.Hash) (err error)
	
	// Get Utxo address and balance, or "", 0 if not stored
	Get(out TxOutId) (data TxOutData, err error)
	
	// Store new utxo
	Set(out primitives.TxOut) (err error)

	// Get all address utxout
	GetByAddress(address string) (outs []primitives.TxOut, err error)

	// Get address accumulated balance 
	GetBalance(address string) (balance int64, err error)
	
	// Remove utxo from storage, if it doesn't exist no error is returned.
	Delete(out TxOutId) (err error)

	// Returns true if Storage contains utxo
	Contains(out TxOutId) (bool, error)

	// Atomic bulk get 
	BulkGet(outs []TxOutId) (data []TxOutData, err error)

	// Atomic bulk utxo insertion and deletion
	BulkUpdate(insert []primitives.TxOut, remove []TxOutId, height int64, hash chainhash.Hash) (err error)

	// Same as Bulk update but using same maps as cache
	BulkUpdateFromMap(insert map[TxOutId]TxOutData, remove map[TxOutId]bool, height int64, hash chainhash.Hash) error

	// Close storage
	Close() error

	// CleanUp wasted space, defrag, vacuum, .... in general recover wasted space if applicable
	CleanUp() error

	// Mark database as dirty, once a databse is marked dirty the only allowed operation is to close it
	// all the other will return ErrDirtyStorage error.
	MarkDirty(reason string) error
}


var (
	ErrNegativeUtxo     = errors.New("Storage: utxo has negative value")
	ErrUnexpendableUtxo = errors.New("Storage: unexpendable utxo")
	ErrNegativeHeight   = errors.New("Storage: Negative height")
	ErrDirtyStorage     = errors.New("Storage: Dirty storage")
)

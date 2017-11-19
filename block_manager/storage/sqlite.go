package storage

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/secnot/gobalance/primitives"
)


type utxo struct {
	TxHash []byte          `db:"tx"`   // Hash for the transaction containing the TxOut
	Value  int64           `db:"value"`// Output ammount
	Addr   string          `db:"addr"` // Bitcoin address from pkScript
	Nout   uint32          `db:"nout"` // Output number
}

var SCHEMAS = [...]string {
	`utxo 	(tx BLOB NOT NULL, 
			 nout integer NOT NULL,
			 addr text NOT NULL,
			 value integer NOT NULL,
			 PRIMARY KEY(tx, nout))`,
	`last_block (pk integer NOT NULL,
			 height integer NOT NULL,
			 hash BLOB NOT NULL,
			 PRIMARY KEY(pk))`,
}

var PRAGMAS = [...]string {	
	"PRAGMA page_size=4096",
	"PRAGMA cache_size=-100000", // 100MB Cache
	"PRAGMA locking_mode=EXCLUSIVE",
	"PRAGMA auto_vacuum=NONE", // NONE, INCREMENTAL, FULL
	"PRAGMA synchronous=NORMAL", // NORMAL
	"PRAGMA temp_store=2", // MEMORY (FILE is 1)
	"PRAGMA journal_mode=TRUNCATE", // WAL, TRUNCATE, MEMORY
}

var TRIGGERS = [...]string {	
	// Raise error when unexpendable utxout is inserted
	`CREATE TRIGGER IF NOT EXISTS Delete_Unexpendable_Utxo
	 BEFORE INSERT ON utxo
	 for each row when new.value = 0 or new.addr = "" begin
	 	SELECT RAISE(ABORT, 'Unexpendable utxo');
	 end`,

	// Raise error when utxo value is negative
    `CREATE TRIGGER IF NOT EXISTS Error_Negative_Utxo
	 BEFORE INSERT ON utxo
	 for each row when new.value < 0 begin
	 	SELECT RAISE(ABORT, 'Negative utxo');
	 end`,
}

var INDEXES = [...]string {
	`Utxo_Addr_Idx ON utxo(addr)`,
}

// InitDB: Opens or creates a SQLite DB and creates missing tables 
func initDB(driverName string, dataSource string) (db *sqlx.DB, err error){

	// The same as the built-in database/sql
	db, err = sqlx.Open(driverName, dataSource)
	if err != nil {
		return
	}

	// Create tables
	for _, schema := range SCHEMAS {
		create_sql := `CREATE TABLE if not exists `
		_, err = db.Exec(create_sql+schema+";")
		if err != nil {
			return nil, err
		}
	}

	// Load pragmas
	for _, pragma := range PRAGMAS {
		_, err := db.Exec(pragma)
		if err != nil {
			return nil, err
		}
	}
	
	// Creat triggers
	for _, trigger := range TRIGGERS {
		_, err = db.Exec(trigger)
		if err != nil {
			return nil, err
		}
    }

    // Create indices
    for _, index := range INDEXES {
        index_sql := `CREATE INDEX IF NOT EXISTS `
        _, err = db.Exec(index_sql+index+";")
        if err != nil {
            return nil, err
        }
    }

	return db, nil
}

type SQLiteStorage struct {	
	//
	db *sqlx.DB

	// Stored statements initialized when the db is openned
	lenStmt *sqlx.Stmt
	getStmt *sqlx.Stmt
	setStmt *sqlx.Stmt
	containsStmt *sqlx.Stmt
	deleteStmt *sqlx.Stmt

	// Get statement with default value for empty
	defaultGetStmt *sqlx.Stmt

	// Get all the txout for a given address
	getByAddressStmt *sqlx.Stmt

	// Get accumulated address balance
	getBalanceStmt *sqlx.Stmt

	// Height related statements
	setLastBlockStmt *sqlx.Stmt
	getLastBlockStmt *sqlx.Stmt
}



// NewSQLiteStorage creates and initializes a new storage
func NewSQLiteStorage(DBPath string) (*SQLiteStorage, error) {

	//TODO: Close DB if there is any error
	//
	db, err := initDB("sqlite3", DBPath)
	if err != nil {
		return nil, err
	}

	// Configure db
	db.SetMaxOpenConns(1)

	store := &SQLiteStorage {
		db: db,
	}
	
	// Create prepared statements
	store.lenStmt, err = db.Preparex("SELECT count(*) FROM utxo;")
	if err != nil {
		return nil, err
	}

	store.getStmt, err = db.Preparex("SELECT addr, value FROM utxo WHERE tx=? AND nout=?;")
	if err != nil {
		return nil, err
	}
	
	store.containsStmt, err = db.Preparex("SELECT value FROM utxo WHERE tx=? AND nout=?;")
	if err != nil {
		return nil, err
	}
	
	store.setStmt, err = db.Preparex("INSERT INTO utxo(tx, nout, addr, value) VALUES(?, ?, ?, ?);")
	if err != nil {
		return nil, err
	}
	
	store.deleteStmt, err = db.Preparex("DELETE FROM utxo WHERE tx=? AND nout=?;")
	if err != nil {
		return nil, err
	}

	store.getLastBlockStmt, err = db.Preparex("SELECT height, hash FROM last_block WHERE pk=1;")
	if err != nil {
		return nil, err
	}

	store.setLastBlockStmt, err = db.Preparex("INSERT OR REPLACE INTO last_block(pk, height, hash) VALUES(1, ?, ?);")
	if err != nil {
		return nil, err
	}

	store.getByAddressStmt, err = db.Preparex("SELECT tx, nout, addr, value FROM utxo WHERE addr=?;")
	if err != nil {
		return nil, err
	}

	store.getBalanceStmt, err = db.Preparex("SELECT coalesce(SUM(value), 0) FROM utxo WHERE addr=?;")
	if err != nil {
		return nil, err
	}

	return store, nil
}


// Set address balance if the address exists it's modified, otherwise it's inserted
func (s *SQLiteStorage) Len() (length int, err error) {
	err = s.lenStmt.QueryRowx().Scan(&length)
	return
}

// GetLastBlock returns the height and hash for the last block committed
func (s *SQLiteStorage) GetLastBlock() (height int64, hash chainhash.Hash, err error) {
	var bHash []byte

	err = s.getLastBlockStmt.QueryRowx().Scan(&height, &bHash)	
	switch { 
	case err == sql.ErrNoRows:
		return -1, primitives.ZeroHash, nil

	case err != nil:
		return -1, primitives.ZeroHash, err

	default:
		copy(hash[:], bHash)
		return height, hash, nil
	}
}

// SetLastBlock sets new last block deleting previous one
func (s *SQLiteStorage) SetLastBlock(height int64, hash chainhash.Hash) (err error) {
	if height <0 {
		return ErrNegativeHeight
	}

	_, err = s.setLastBlockStmt.Exec(height, hash[:])
	return
}

// Set stores a new utxo record
func (s *SQLiteStorage) Set(out primitives.TxOut) (err error) {
	_, err = s.setStmt.Exec(out.TxHash[:], out.Nout, out.Addr, out.Value)
	if err == nil {
		return
	}

	if err.Error() == "Negative utxo" {
		err = ErrNegativeUtxo
	} else if err.Error() == "Unexpendable utxo" {
		err = ErrUnexpendableUtxo
	}

	return
}

// Get return TxOutData
func (s *SQLiteStorage) Get(out TxOutId) (data TxOutData, err error) {
	err = s.getStmt.QueryRowx(out.TxHash[:], out.Nout).StructScan(&data)
	switch {
	// If not present return default value
	case err == sql.ErrNoRows:
		err = nil
		data = TxOutData{Addr:"", Value: 0}
	}
	return 
}

// GetByAddress returns wallet's unexpent txouts
func (s *SQLiteStorage) GetByAddress(address string) (outs []primitives.TxOut, err error) {
	var utxos []utxo
	err = s.getByAddressStmt.Select(&utxos, address)
	if err != nil {
		return nil, err
	}

	txouts := make([]primitives.TxOut, len(utxos))
	for n, out := range utxos {
		var hash chainhash.Hash
		copy(hash[:], out.TxHash) 
		txouts[n] = primitives.TxOut {
			TxHash: &hash,
			Nout: out.Nout,
			Addr: out.Addr,
			Value: out.Value,
		}
	}

	return txouts[:], nil
}

// GetBalance returns address balance
func (s *SQLiteStorage) GetBalance(address string) (balance int64, err error) {
	err = s.getBalanceStmt.QueryRow(address).Scan(&balance)
	if err != nil {
		return -1, err
	}
	outs, _ := s.GetByAddress(address)
	sum := int64(0)
	for _, out := range outs {
		sum += out.Value
	}
	return balance, err
}

// Contains returns true if the db contains the address
func (s *SQLiteStorage) Contains(out TxOutId) (bool, error) {
	var value int64
	err := s.containsStmt.QueryRow(out.TxHash[:], out.Nout).Scan(&value)

	switch { 
	case err == sql.ErrNoRows:
		return false, nil

	case err != nil:
		return false, err

	default:
		return true, nil
	}

}

// Remove Utxo from storage
func (s *SQLiteStorage) Delete(out TxOutId) (err error) {
	_, err = s.deleteStmt.Exec(out.TxHash, out.Nout)
	return
}


// Transaction wrapper
func Transact(db *sqlx.DB, txFunc func(*sqlx.Tx) error) (err error) {
    tx, err := db.Beginx()
    if err != nil {
        return
    }
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p) // re-throw panic after Rollback
        } else if err != nil {
            tx.Rollback()
        } else {
            err = tx.Commit()
        }
    }()
    err = txFunc(tx)
    return err
}

// BulkGet utxo get WITHOUT DEFAULT
func (s *SQLiteStorage) BulkGet(outs []TxOutId) (data []TxOutData, err error) {
	if len(outs) == 0 {
		return nil, nil
	}

	data = make([]TxOutData, len(outs)) 
	err = Transact(s.db, func(tx *sqlx.Tx) error {
		getStmt := tx.Stmtx(s.getStmt)
		for n, out := range outs {
			err := getStmt.QueryRowx(out.TxHash[:], out.Nout).StructScan(&data[n])
			
			switch {
			// If not present return default value
			case err == sql.ErrNoRows:
				err = nil
				data[n].Addr  = ""
				data[n].Value = 0

			case err != nil:
				return err
			}	
		}
		return nil
	})

	if err != nil {
		data = nil 
	}
	return
}


// BulkUpdate Atomic bulk storage update
func (s *SQLiteStorage) BulkUpdate(insert []primitives.TxOut, remove []TxOutId, height int64, hash chainhash.Hash) (err error) {
	if height < 0 {
		return ErrNegativeHeight
	}

	err = Transact(s.db, func(tx *sqlx.Tx) error {
		setStmt       := tx.Stmtx(s.setStmt)
		deleteStmt    := tx.Stmtx(s.deleteStmt)
		lastBlockStmt := tx.Stmtx(s.setLastBlockStmt)

		// Delete expent utxo
		for _, rem := range remove {
			_, err = deleteStmt.Exec(rem.TxHash[:], rem.Nout)
			if err != nil {
				return err
			}
		}

		// Insert new utxo
		for _, ins := range insert {
			_, err = setStmt.Exec(ins.TxHash[:], ins.Nout, ins.Addr, ins.Value)
			if err != nil {	
				if err.Error() == "Negative utxo" {
					err = ErrNegativeUtxo
				} else if err.Error() == "Unexpendable utxo" {
					err = ErrUnexpendableUtxo
				}
				return err
			}
		}

		// Set lastblock
		_, err = lastBlockStmt.Exec(height, hash[:])
		return err
	})

	return err
}



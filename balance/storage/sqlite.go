package storage

import (
	"fmt"
	"database/sql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	SQLITE_VARIABLE_LIMIT = 999
)

var schemas = [...]string{
	`balance (address text NOT NULL, 
			  balance integer NOT NULL,
			  PRIMARY KEY(address))`,
	`height (pk integer NOT NULL,
			 height integer NOT NULL,
			 PRIMARY KEY(pk))`,
}


type SQLiteStorage struct {
	
	//
	db *sqlx.DB

	// Stored statements initialized when the db is openned
	lenStmt *sqlx.Stmt
	getStmt *sqlx.Stmt
	setStmt *sqlx.Stmt
	updateStmt *sqlx.Stmt
	containsStmt *sqlx.Stmt
	deleteStmt *sqlx.Stmt

	setHeightStmt *sqlx.Stmt
	getHeightStmt *sqlx.Stmt
}


// InitTables initializes missing tables
func initTable(db *sqlx.DB, schema string) (err error) {
	create_sql := `CREATE TABLE if not exists `
	
	_, err = db.Exec(create_sql+schema+";")
	return err
}

// InitDB: Opens a DB and creates missing tables 
func initDB(driverName string, dataSource string) (db *sqlx.DB, err error){

	// The same as the built-in database/sql
	db, err = sqlx.Open(driverName, dataSource)
	if err != nil {
		return
	}

	// Initialize missing tables
	for _, schema := range schemas {
		err = initTable(db, schema)
		if err != nil {
			return nil, err
		}
	}

	// Add trigger to delete empty balances
	_, err = db.Exec(`CREATE TRIGGER IF NOT EXISTS Delete_Zero_Trigger 
			 AFTER INSERT ON balance
			 for each row when new.balance = 0 begin
			     delete from balance where address = new.address;
			 end`)
	if err != nil {
		return nil, err
	}

	// Raise error when balance goes below zero
    _, err = db.Exec(`CREATE TRIGGER IF NOT EXISTS Error_Negative_Balance
            BEFORE INSERT ON balance
            for each row when new.balance < 0
            BEGIN
                SELECT RAISE(ABORT, 'Negative balance');
            END`)
    if err != nil {
        return nil, err
    }

	return db, nil
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
	db.Exec("PRAGMA page_size=4096")
	db.Exec("PRAGMA cache_size=10000")
	db.Exec("PRAGMA temp_store=MEMORY")
	db.Exec("PRAGMA journal_mode=MEMORY") // TODO: NOT SAFE


	store := &SQLiteStorage {
		db: db,
	}
	
	// Create prepared statements
	store.lenStmt, err = db.Preparex("SELECT count(*) FROM balance;")
	if err != nil {
		return nil, err
	}

	store.getStmt, err = db.Preparex("SELECT balance FROM balance WHERE address=?;")
	if err != nil {
		return nil, err
	}
	
	store.containsStmt, err = db.Preparex("SELECT balance FROM balance WHERE address=?;")
	if err != nil {
		return nil, err
	}
	
	store.setStmt, err = db.Preparex("INSERT OR REPLACE INTO balance(address, balance) VALUES(?, ?);")
	if err != nil {
		return nil, err
	}
	
	store.updateStmt, err = db.Preparex(`INSERT OR REPLACE INTO balance (address, balance) 
		VALUES ($1,
				COALESCE( (SELECT balance FROM balance WHERE address=$1), 0) + $2)`)
	if err != nil {
		return nil, err
	}

	store.deleteStmt, err = db.Preparex("DELETE FROM balance WHERE address=?;")
	if err != nil {
		return nil, err
	}

	store.getHeightStmt, err = db.Preparex("SELECT height FROM height WHERE pk=1;")
	if err != nil {
		return nil, err
	}
	
	store.setHeightStmt, err = db.Preparex("INSERT OR REPLACE INTO height(pk, height) VALUES(1, ?);")
	if err != nil {
		return nil, err
	}

	return store, nil
}


// Set address balance if the address exists it's modified, otherwise it's inserted
func (s *SQLiteStorage) Len() (length int, err error) {
	err = s.lenStmt.QueryRow().Scan(&length)
	return
}

// Set address balance
func (s *SQLiteStorage) Set(address string, balance int64) (err error) {
	_, err = s.setStmt.Exec(address, balance) // Trigger will delete if balance = 0
	if err != nil && err.Error() == "Negative balance" {
		errMsg := fmt.Sprintf("%v: %v", address, balance)
		return NewNegativeBalanceError(errMsg)
	}
	return
}

// Update address balance by adding or substracting a balue
func (s *SQLiteStorage) Update(address string, update int64) (err error) {
	_, err = s.updateStmt.Exec(address, update) // Trigger will delete if update+balance = 0
	if err != nil && err.Error() == "Negative balance" {
		errMsg := fmt.Sprintf("%v: %v", address, update)
		return NewNegativeBalanceError(errMsg)
	}
	return
}

// Get address balance or 0 if it isn't stored
func (s *SQLiteStorage) Get(address string) (value int64, err error) {
	err = s.getStmt.QueryRow(address).Scan(&value)
	switch { 
	case err == sql.ErrNoRows:
		return 0, nil

	case err != nil:
		return 0, err

	default:
		return value, nil
	}
}

// Delete address balance from storage, if the address
// doesn't exist no error is returned.
func (s *SQLiteStorage) Delete(address string) (err error) {
	_, err = s.deleteStmt.Exec(address)
	return
}

// GetHeight gets current height, return error if none set
func (s *SQLiteStorage) GetHeight() (height int64, err error) {
	err = s.getHeightStmt.QueryRow().Scan(&height)
	
	switch { 
	case err == sql.ErrNoRows:
		return -1, nil

	case err != nil:
		return -1, err

	default:
		return height, nil
	}
}


// SetHeight sets current height
func (s *SQLiteStorage) SetHeight(height int64) (err error) {
	_, err = s.setHeightStmt.Exec(height)
	return
}

// Contains returns true if the db contains the address
func (s *SQLiteStorage) Contains(address string) (bool, error) {
	var value int64
	err := s.containsStmt.QueryRow(address).Scan(&value)

	switch { 
	case err == sql.ErrNoRows:
		return false, nil

	case err != nil:
		return false, err

	default:
		return true, nil
	}

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


// 
func (s *SQLiteStorage) batchBulkGet(address[]string, balance map[string]int64) (err error) {
	if len(address) == 0 {
		return nil
	}
	
	q, args, err := sqlx.In("SELECT address, ifnull(balance, 0) FROM balance WHERE address IN(?);", address)
	if err != nil {
		return err
	}
	
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return err
	}

	// Scan results
	var addr string
	var bal int64
	for rows.Next() {
		if err = rows.Scan(&addr, &bal); err != nil {
			break
		}
		balance[addr] = bal
	}
	return err
}

// Bulk 
func (s *SQLiteStorage) BulkGet(addresses []string) ([]int64, error) {
	if len(addresses) == 0 {
		return nil, nil
	}
	//TODO: Use transact to wrap all batches 
	// Split address into enough queries so SQLite3 SQLITE_VARIABLE_LIMIT
	// isn't reached.
	balance := make(map[string]int64, len(addresses))
	for i:=0; i<len(addresses); i += SQLITE_VARIABLE_LIMIT {
		batch_end := min(i+SQLITE_VARIABLE_LIMIT, len(addresses))
		err := s.batchBulkGet(addresses[i:batch_end], balance)
		if err != nil {
			return nil, err
		}
	}

	// Sort balance in addresses order
	sorted_balance := make([]int64, len(addresses))
	for n, addr := range addresses {
		sorted_balance[n] = balance[addr]
	}
	return sorted_balance, nil
}


// BulkUpdate Atomic bulk storage update
func (s *SQLiteStorage) BulkUpdate(updates []AddressBalancePair, 
		height int64) (err error) {
	
	// Start transaction
	err = Transact(s.db, func (tx *sqlx.Tx) error {
	
		updateStmt := tx.Stmtx(s.updateStmt)
		heightStmt := tx.Stmtx(s.setHeightStmt)
		
		for _, up := range updates {
			if up.Balance == 0 {
				continue
			}
			if _, err = updateStmt.Exec(up.Address, up.Balance); err != nil {
				return err
			}
		}

		if _, err = heightStmt.Exec(height); err != nil {
			return err
		}
		return nil
	})

	if err != nil && err.Error() == "Negative balance" {
		errMsg := fmt.Sprintf("Negative balance ant height %v", height)
		return NewNegativeBalanceError(errMsg)
	}
	return err
}























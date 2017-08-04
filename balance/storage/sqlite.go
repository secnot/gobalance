package storage

import (
	"log"
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

	// Simple stored statements
	lenStmt *sqlx.Stmt
	getStmt *sqlx.Stmt
	setStmt *sqlx.Stmt
	containsStmt *sqlx.Stmt
	deleteStmt *sqlx.Stmt

	setHeightStmt *sqlx.Stmt
	getHeightStmt *sqlx.Stmt
	
	// Bulk stored statements
	bulkGetStmt *sqlx.Stmt
	bulkInsertStmt *sqlx.Stmt
	bulkUpdateStmt *sqlx.Stmt
	bulkDeleteStmt *sqlx.Stmt

	//
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
	//db.Exec("PRAGMA SQLITE_STATIC")


	store := &SQLiteStorage {
		db: db,
	}
	
	// Add prepared statements
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

	store.bulkGetStmt, err = db.Preparex("SELECT coalesce(balance, 0) FROM balance WHERE address=?;")
	if err != nil {
		return nil, err
	}
	
	store.bulkInsertStmt, err = db.Preparex("INSERT OR REPLACE INTO balance(address, balance) VALUES (?, ?);")
	if err != nil {
		return nil, err
	}
	
	store.bulkUpdateStmt, err = db.Preparex("INSERT OR REPLACE INTO balance(address, balance) VALUES (?, ?);")
	if err != nil {
		return nil, err
	}

	store.bulkDeleteStmt, err = db.Preparex("DELETE FROM balance WHERE address=?;")
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
	if balance != 0 {
		_, err = s.setStmt.Exec(address, balance)
	} else {
		err = s.Delete(address)
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
// doesn't exist not error is returned.
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

// BulkGet Atomic bulk balance get
// https://stackoverflow.com/questions/20271123/how-to-execute-an-in-lookup-in-sql-using-golang
// https://stackoverflow.com/questions/20271123/how-to-execute-an-in-lookup-in-sql-using-golang
func (s *SQLiteStorage) oldBulkGet(addresses []string) ([]int64, error) {
	//SQLITE_VARIABLE_LIMIT = 999

	if len(addresses) == 0 {
		return nil, nil
	}

	//q, args, err := sqlx.In("SELECT ifnull(balance, 0) FROM balance WHERE address IN(?);", addresses)
	q, args, err := sqlx.In("SELECT address, ifnull(balance, 0) FROM balance WHERE address IN(?);", addresses)
	if err != nil {
		return nil, err
	}
	
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}

	// Scan results
	balance := make(map[string]int64, len(addresses))
	var addr string
	var bal int64
	for rows.Next() {
		err = rows.Scan(&addr, &bal)
		if err != nil {
			return nil, err
		}
		balance[addr] = bal
	}
	rows.Close()

	// Sort results matching addresses order
	sorted_balance := make([]int64, len(addresses))
	for n, addr := range addresses {
		sorted_balance[n] = balance[addr]
	}
	return sorted_balance, nil
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

// TODO: Add SQLITE_VARIABLE_LIMIT tests
func (s *SQLiteStorage) BulkGet(addresses []string) ([]int64, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

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
// TODO: User real bulk inserts https://stackoverflow.com/questions/21108084/golang-mysql-insert-multiple-data-at-once
// TODO: About transactions https://stackoverflow.com/questions/16184238/database-sql-tx-detecting-commit-or-rollback
func (s *SQLiteStorage) BulkUpdate(insert []AddressBalancePair, 
			   update []AddressBalancePair, 
			   remove []string, height int64) (err error) {
	
	// Start transaction
	tx, err := s.db.Beginx()
	insertStmt := tx.Stmtx(s.bulkInsertStmt)
	updateStmt := tx.Stmtx(s.bulkUpdateStmt)
	removeStmt := tx.Stmtx(s.bulkDeleteStmt)
	heightStmt := tx.Stmtx(s.setHeightStmt)

	for _, pair := range insert {
		if pair.Balance != 0 {
			_, err = insertStmt.Exec(pair.Address, pair.Balance)
			if err != nil {
				log.Print(err)
				tx.Rollback()
				return err
			}
		}
	}

	for _, pair := range update {
		if pair.Balance != 0 {
			_, err = updateStmt.Exec(pair.Address, pair.Balance)
		} else {
			_, err = removeStmt.Exec(pair.Address)
		}
		if err != nil {
			log.Print(err)
			tx.Rollback()
			return err
		}
	}
	
	for _, addr := range remove {
		
		_, err = removeStmt.Exec(addr)
		if err != nil {
			log.Print(err)
			tx.Rollback()
			return err
		}
	}

	_, err = heightStmt.Exec(height)
	if err != nil {
		log.Print(err)
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	return err
}























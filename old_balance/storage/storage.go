package storage


type AddressBalancePair struct {
	Address string
	Balance int64
}

// Memory and SQL storage interface:
type Storage interface {

	// Number of stored addresses
	Len() (length int, err error)
	
	// Get current height, return -1 if none stored
	GetHeight() (height int64, err error)

	// Set New height
	SetHeight(height int64) (err error)

	// Get address balance or 0 if it isn't stored
	Get(address string) (value int64, err error)

	// Set address balance overwritting current balance if it exists.
	Set(address string, value int64) (err error)

	// Update or create an address balance by adding or substracting a 
	// value. If the resulting balance is 0 the record is deleted.
	Update(address string, value int64) (err error)

	// Delete address balance from storage, if the address
	// doesn't exist not error is returned.
	Delete(address string) (err error)

	// Returns true if Storage contains address
	Contains(address string) (bool, error)

	// Atomic bulk balance query
	BulkGet(address []string) (balance []int64, err error)

	// Atomic bulk balance update
	BulkUpdate(update []AddressBalancePair, height int64) (err error)
}





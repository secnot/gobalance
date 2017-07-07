package balance


// Memory and SQL storage interface:
type Storage interface {

	// Number of stored addresses
	Len() (length int, err error)
	
	// Get current height, return error if none set
	GetHeight() (height int64, err error)

	// Set New height
	SetHeight(height int64) (err error)

	// Get address balance
	Get(address string) (value int64, err error)

	// Set address balance
	Set(address string, value int64) (err error)

	// Delete address balance from storage, if the address
	// doesn't exist not error is returned.
	Delete(address string) (err error)

	// Returns true if Storage contains address
	Contains(address string) (bool, error)

	// Atomic bulk balance get
	BulkGet(addresses []string) (balance []int64, err error)

	// Atomic bulk storage update
	BulkUpdate(insert []AddressBalancePair, 
			   update []AddressBalancePair, 
			   remove []string, height int64) (err error)
}


type AddressBalancePair struct {
	Address string
	Balance int64
}



type ErrorStorage struct {
	message string
}

func NewErrorStorage(message string) *ErrorStorage {
	return &ErrorStorage{
		message: message,
	}
}

func (e *ErrorStorage) Error() string {
	return e.message
}



// Operations would result in a negative balance
type ErrorNegativeBalance struct {
	message string
}

func NewErrorNegativeBalance(message string) *ErrorNegativeBalance {
	return &ErrorNegativeBalance{
		message: message,
	}
}

func (e *ErrorNegativeBalance) Error() string {
	return e.message
}

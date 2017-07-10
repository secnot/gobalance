package storage


// StorageError returned when it wasn't possible to read/write to storage
type StorageError struct {
	Message string
}

func NewStorageError(message string) *StorageError {
	return &StorageError{
		Message: message,
	}
}

func (e StorageError) Error() string {
	return e.Message
}



// Operations resulting in a negative balance
type NegativeBalanceError struct {
	Message string
}

func NewNegativeBalanceError(message string) *NegativeBalanceError {
	return &NegativeBalanceError{
		Message: message,
	}
}

func (e NegativeBalanceError) Error() string {
	return e.Message
}

package balance

import (
	"sync"
)


type BalanceProcessor struct {

	// Concurrency lock
	sync.RWMutex

}

func NewBalanceProcessor() (balance *BalanceProcessor) {
	return nil
}




type Balance struct {

	// Only blocked
	sync.RWMutex

	// Current block height
	Height uint64 

}


func (b *Balance) AddBlock() {
}


// Backtrack a single block
func (b *Balance) Backtrack() (err error) {
	return nil
}

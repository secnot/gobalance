package height

import (
	"sync"
	"github.com/secnot/gobalance/block_manager"
)



var Height uint64 = 0 
var Lock sync.RWMutex

func HeightRoutine() {	

	updateChan := block_manager.Subscribe(10)
	
	for update := range updateChan{

		Lock.Lock()
		switch update.Class {
		case block_manager.OP_NEWBLOCK:
			Height = update.Block.Height
		case block_manager.OP_BACKTRACK:
			Height = update.Block.Height - 1
		}
		Lock.Unlock()
	}
}


func GetHeight() (height uint64) {
	Lock.RLock()
	height = Height
	Lock.RUnlock()
	return
}

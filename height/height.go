package height

import (
	"sync"
	"github.com/secnot/gobalance/crawler"
)



var Height uint64 = 0 
var Lock sync.RWMutex

func HeightRoutine() {	

	updateChan := crawler.Subscribe(10)
	
	for update := range updateChan{

		Lock.Lock()
		switch update.Class {
		case crawler.OP_NEWBLOCK:
			Height = update.Block.Height
		case crawler.OP_BACKTRACK:
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

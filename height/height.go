package height

import (
	"sync"
	"github.com/secnot/gobalance/block_manager"
)



type HeightCache struct {
	sync.RWMutex
	manager *block_manager.BlockManager
	height uint64

	//
	stopChan chan chan bool
}

// NewHeightCache initializes and starts cache
func NewHeightCache(manager *block_manager.BlockManager) *HeightCache {
	
	cache := &HeightCache {
		stopChan: make(chan chan bool),
		manager:  manager,
		height:   0,
	}

	go cache.heightRoutine()
	return cache
}

// heightRoutine handles block updades and stop signal
func (h *HeightCache) heightRoutine() {	

	updateChan := h.manager.Subscribe(10)
	
	for {

		select {
		case update := <- updateChan:
			h.Lock()
			switch update.Class {
			case block_manager.OP_NEWBLOCK:
				h.height = update.Block.Height
			case block_manager.OP_BACKTRACK:
				h.height = update.Block.Height - 1
			}
			h.Unlock()

		case ch := <- h.stopChan:
			h.manager.Unsubscribe(updateChan)
			ch <- true
			
			return
		}
	}
}

// Stop sends signal and waits for confirmation
func (h *HeightCache) Stop() {
	doneCh := make(chan bool)
	h.stopChan <- doneCh
	<-doneCh
	close(doneCh)
}

// GetHeight for the current top of the chain
func (h *HeightCache) GetHeight() (height uint64) {
	h.RLock()
	height = h.height
	h.RUnlock()
	return
}

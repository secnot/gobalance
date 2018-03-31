package block_manager

import (
	"log"
	"github.com/secnot/gobalance/interfaces"
)


func Logger(manager *BlockManager) {
	blocks := manager.Subscribe(10) 

	for update := range blocks {
		block := update.Block
		switch update.Class  {
		case interfaces.OP_NEWBLOCK:
			if block.Height % 1000 == 0 {
				log.Printf("New: %v\n", block)
			}
		case interfaces.OP_BACKTRACK:
			log.Printf("Backtrack: %v\n", block)
		
		case interfaces.OP_COMMIT:
			log.Printf("Commit: started\n")

		case interfaces.OP_COMMIT_DONE:
			log.Printf("Commit: finished\n")
		}
	}
}

package block_manager

import (
	"log"
)


func Logger(manager *BlockManager) {
	blocks := manager.Subscribe(10) 

	for update := range blocks {
		block := update.Block
		if update.Class == OP_NEWBLOCK {
			if block.Height % 1000 == 0 {
				log.Printf("New: %v\n", block)
			}
		} else {
			log.Printf("Backtrack: %v\n", block)
		}
	}
}

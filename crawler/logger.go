package crawler

import (
	"log"
)


func Logger() {
	blocks := Subscribe(10) 

	for update := range blocks {
		block := update.Block
		if update.Class == OP_NEWBLOCK {
			log.Printf("New: %v\n", block)
		} else {
			log.Printf("Backtrack: %v\n", block)
		}
	}
}

package logger

import (
	"log"
	"github.com/secnot/gobalance/primitives"
)


type Logger struct {}

// NewLogger Creates and initializes logging observer
func NewLogger() *Logger {
	return &Logger{}	
}


// NewBlock new block CrawlerObserver interface method
func (c *Logger) NewBlock(block *primitives.Block) {
	log.Printf("New: (%v) %v\n", block.Height, block.Hash)
}

// BacktrackBlock backtrack block CrawlerOberserver interface method
func (c *Logger) BacktrackBlock(block *primitives.Block) {
	log.Printf("Backtrack: (%v) %v\n", block.Height, block.Hash)
}

// Crawler adapter that waits until a block has certain number of confirmations
// before sending it to the subscribers.
package crawler
	
import (
	"sync"
	"github.com/phf/go-queue/queue"
	"github.com/secnot/gobalance/primitives"
)


// ConfirmedAdapter waits until a block has been confirmed by the required number
// of blocks before relaying new blocks
type ConfirmedAdapter struct {
	sync.Mutex

	// Last N blocks (type: primitives.Block)
	blocks *queue.Queue

	// Required confirmations before relaying a block
	confirmations uint32
	
	// Callback 
	subscribers []CrawlerObserver
}


// NewConfirmedAdapter
func NewConfirmedAdapter(confirmations uint32) *ConfirmedAdapter {
	return &ConfirmedAdapter{
		confirmations: confirmations,
		blocks: queue.New(),
	}
}

// NewBlock 
func (c *ConfirmedAdapter) NewBlock(block *primitives.Block) {
	c.Lock()
	c.blocks.PushBack(block)
	if c.blocks.Len() > int(c.confirmations) {
		confirmed := c.blocks.PopFront().(*primitives.Block)
		for _, sub := range c.subscribers {
			sub.NewBlock(confirmed)
		}
	}
	c.Unlock()
}

// BacktrackBlock
func (c *ConfirmedAdapter) BacktrackBlock(block *primitives.Block) {
	c.Lock()
	if c.blocks.Len() == 0 {	
		for _, sub := range c.subscribers {
			sub.BacktrackBlock(block)
		}
	} else {
		c.blocks.PopBack()
	}
	c.Unlock()
}

// Subscribe
func (c *ConfirmedAdapter) Subscribe(subs CrawlerObserver) {
	c.Lock()
	c.subscribers = append(c.subscribers, subs)
	c.Unlock()
}

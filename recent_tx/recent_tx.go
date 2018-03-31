package recent_tx

import (
	"github.com/secnot/gobalance/interfaces"
	"github.com/secnot/gobalance/primitives"
)

type TxResponse struct {
	Tx []*primitives.Tx
	Block []*primitives.Block
}

type TxRequest struct {
	Address string
	Response chan TxResponse
}


type RecentTxCache struct {

	// max number of blocks cached
	size uint16

	// 
	queue *primitives.BlockQueue

	// block manager
	manager interfaces.BlockManager
	
	// transaction requests
	requestChan chan TxRequest

	// signal to stop cache
	stopChan chan chan bool
}


// NewRecentTxCache 
func NewRecentTxCache(manager interfaces.BlockManager, trackedBlocks uint16) *RecentTxCache {
	cache := &RecentTxCache{
		requestChan: make(chan TxRequest),
		stopChan:    make(chan chan bool),
		manager:     manager,
		queue:       primitives.NewBlockQueue(),
		size:        trackedBlocks,
	}

	go cache.recentTxRoutine()
	return cache
}

// backtrackBlock
func (r *RecentTxCache) backtrackBlock(block *primitives.Block) {
	r.queue.PopBack()
}

// newBlock
func (r *RecentTxCache) newBlock(block *primitives.Block) {
	r.queue.PushBack(block)
	if r.queue.Len() > int(r.size) {
		r.queue.PopFront()
	}
}


// GetAddrRecentTx returns the transactions with in the cached blocks that 
// cointaned the address, together with the blocks containing thos transactions
func (r *RecentTxCache) getAddrRecentTx(address string) ([] *primitives.Tx, [] *primitives.Block) {

	// TODO: Use getPeer when in commit mode.
	transactions := r.queue.GetTx(address)

	// Get the blocks containing each transaction
	blocks := make([]*primitives.Block, len(transactions))
	for n, tx := range transactions {
		_, block := r.queue.Tx(*tx.Hash)
		blocks[n] = block
	}

	return transactions, blocks
}



// RecentTx is the routine handling block updates and client requests
func (r *RecentTxCache) recentTxRoutine() {
	
	//
	updatesChan  := r.manager.Subscribe(10)

	//proxy := false
	for {

		select {
		case update := <- updatesChan:
			
			switch update.Class {
			case interfaces.OP_NEWBLOCK:
				r.newBlock(update.Block)
			case interfaces.OP_BACKTRACK:
				r.backtrackBlock(update.Block)
			
			/* TODO: Request transactions from proxy when in commit
			case block_manager.OP_COMMIT:
				proxy = true
			case block_manager.OP_COMMIT_DONE:
				proxy = false
			*/
			}

		case request := <- r.requestChan:
			tx, blocks := r.getAddrRecentTx(request.Address)
			request.Response <- TxResponse{Tx: tx, Block: blocks}
		
		case ch := <- r.stopChan:
			close(r.stopChan)
			close(r.requestChan)
			ch <- true
			return
		}
	}
}

func (r *RecentTxCache) Stop() {
	doneCh := make(chan bool)

	r.stopChan <- doneCh
	<- doneCh
	close(doneCh)
}

func (r *RecentTxCache) GetRecentTx(address string) ([]*primitives.Tx, []*primitives.Block, error) {
	responseCh := make(chan TxResponse)
	
	r.requestChan <- TxRequest{Address: address, Response: responseCh}
	response := <- responseCh
	close(responseCh)
	return response.Tx, response.Block, nil
}

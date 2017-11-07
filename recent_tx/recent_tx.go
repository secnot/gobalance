package recent

import (
	"github.com/secnot/gobalance/crawler"
	"github.com/secnot/gobalance/primitives"
)

const (
	// Size for buffered
	TxRequestChanSize = 1
)

var (
	TxRequestChan = make(chan TxRequest, TxRequestChanSize)
)
	

type TxResponse struct {
	Tx []*primitives.Tx
	Block []*primitives.Block
}

type TxRequest struct {
	Address string
	Response chan TxResponse
}


type BlockCache struct {

	// Number of blocks cached
	size int

	// 
	queue *primitives.BlockQueue
}



func NewBlockCache(size int) *BlockCache {
	return &BlockCache{
		size: size,
		queue: primitives.NewBlockQueue(),
	}
}

// Backtrack
func (b *BlockCache) Backtrack(block *primitives.Block) {
	b.queue.PopBack()
}

// NewBlosck
func (b *BlockCache) NewBlock(block *primitives.Block) {
	b.queue.PushBack(block)
	if b.queue.Len() > b.size {
		b.queue.PopFront()
	}
}


// GetAddrRecentTx returns the transactions with in the cached blocks that 
// cointaned the address, together with the blocks containing thos transactions
func (b *BlockCache) GetAddrRecentTx(address string) ([] *primitives.Tx, [] *primitives.Block) {

	transactions := b.queue.GetTx(address)

	// Get the blocks containing each transaction
	blocks := make([]*primitives.Block, len(transactions))
	for n, tx := range transactions {
		_, block := b.queue.Tx(*tx.Hash)
		blocks[n] = block
	}

	return transactions, blocks
}



// RecentTx is the routine handling block updates and client requests
func RecentTxRoutine(cacheSize uint16) {
	
	cache := NewBlockCache(int(cacheSize))

	//
	updatesChan  := crawler.Subscribe(10)

	for {

		select {
		case update := <- updatesChan:
			
			switch update.Class {
			case crawler.OP_NEWBLOCK:
				cache.NewBlock(update.Block)
			case crawler.OP_BACKTRACK:
				cache.Backtrack(update.Block)
			}

		case request := <- TxRequestChan:
			tx, blocks := cache.GetAddrRecentTx(request.Address)
			request.Response <- TxResponse{Tx: tx, Block: blocks}
		}
	}
}

//
func GetRecentTx(address string) ([]*primitives.Tx, []*primitives.Block) {
	responseCh := make(chan TxResponse)
	
	TxRequestChan <- TxRequest{Address: address, Response: responseCh}
	response := <- responseCh
	close(responseCh)
	return response.Tx, response.Block
}

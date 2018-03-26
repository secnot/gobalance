package api_common


type TxOut struct {
	Address string `json:"address"`
	Value int64    `json:"value"`
}

type Tx struct {
	Hash string     `json:"hash"`
	Block *Block 	`json:"block,omitempty"`
	Inputs  []TxOut `json:"inputs,omitempty"`
	Outputs []TxOut `json:"outputs,omitempty"`
}

type Block struct {
	Hash string       `json:"hash"`
	Height int64      `json:"height"`
	Transactions []Tx `json:"transactions,omitempty"`
}

type Address struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
}

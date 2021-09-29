package blockchain

type Block struct {
	Height int
	PrevHash [32]byte
	MerkleRoot [32]byte
	Time int32
	Bits int32
	Nonce int32
	Transactions []Transaction
}

type Transaction struct {
	Flag [2]byte
	Inputs []TxIn
	Outputs []TxOut
	LockTime int32
}

type TxIn struct {
	PrevTxHash [32]byte
	PrevTxIndex int32
	Script []byte
	Sequence [4]byte
}

type TxOut struct {
	Value int64
	ScriptLen int
	Script []byte
}

package blockchain

type VarInt []byte

type Block struct {
	Magic uint32
	Size int32
	Version int32
	PrevHash [32]byte
	MerkleRootHash [32]byte
	Time int32
	Bits int32
	Nonce int32
	TransactionCount VarInt
	Transactions []Transaction
}

type Transaction struct {
	Version int32
	Flag [2]byte
	InCount VarInt
	Inputs []TxIn
	OutCount VarInt
	Outputs []TxOut
	LockTime int32
}

type TxIn struct {
	PrevTxHash [32]byte
	PrevTxIndex int32
	ScriptLen VarInt
	Script []byte
	Sequence [4]byte
}

type TxOut struct {
	Value int64
	ScriptLen VarInt
	Script []byte
}

var Blockchain []Block

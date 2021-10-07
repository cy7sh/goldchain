package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

type Block struct {
	Height int
	Version int
	Hash [32]byte
	PrevHash [32]byte
	MerkleRoot [32]byte
	Time int
	Bits int
	Nonce int
	Transactions []*Transaction
}

type Transaction struct {
	Version int
	Flag [2]uint8
	Inputs []*TxIn
	Outputs []*TxOut
	Witnesses [][]byte
	LockTime int
}

type TxIn struct {
	PrevTxHash [32]byte
	PrevTxIndex int
	Script []byte
	Sequence [4]byte
}

type TxOut struct {
	Value int
	Script []byte
}

func (b *Block) GetHash() [32]byte {
	var headerBuff bytes.Buffer
	binary.Write(&headerBuff, binary.LittleEndian, int32(b.Version))
	headerBuff.Write(b.PrevHash[:])
	headerBuff.Write(b.MerkleRoot[:])
	binary.Write(&headerBuff, binary.LittleEndian, uint32(b.Time))
	binary.Write(&headerBuff, binary.LittleEndian, uint32(b.Bits))
	binary.Write(&headerBuff, binary.LittleEndian, uint32(b.Nonce))
	header := make([]byte, headerBuff.Len())
	headerBuff.Read(header)
	single := sha256.Sum256(header[:])
	double := sha256.Sum256(single[:])
	return double
}

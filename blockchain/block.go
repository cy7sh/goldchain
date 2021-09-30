package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type Block struct {
	Height int
	Hash [32]byte
	PrevHash [32]byte
	MerkleRoot [32]byte
	Time int32
	Bits int32
	Nonce int32
	Transactions []*Transaction
}

type Transaction struct {
	Flag [2]byte
	Inputs []*TxIn
	Outputs []*TxOut
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
	Script []byte
}

func (b *Block) GetHash() [32]byte {
	var headerBuff bytes.Buffer
	binary.Write(&headerBuff, binary.LittleEndian, int32(1))
	headerBuff.Write(b.PrevHash[:])
	headerBuff.Write(b.MerkleRoot[:])
	binary.Write(&headerBuff, binary.LittleEndian, uint32(b.Time))
	binary.Write(&headerBuff, binary.LittleEndian, uint32(b.Bits))
	binary.Write(&headerBuff, binary.LittleEndian, uint32(b.Nonce))
	header := make([]byte, headerBuff.Len())
	headerBuff.Read(header)
	fmt.Printf("header: %x\n", header)
	single := sha256.Sum256(header[:])
	double := sha256.Sum256(single[:])
	return double
}

func byteToLittleEndian(data []byte) []byte {
	s := make([]byte, 0)
	copy(s, data)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
	    s[i], s[j] = s[j], s[i]
	}
	return s
}

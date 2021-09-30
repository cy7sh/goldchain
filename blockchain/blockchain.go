package blockchain

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

var Blockchain []Block
var db *sql.DB
var LastBlock *Block
var rootPath string // where the block chain sould be stored

func Start() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	rootPath = home + "/.goldchain/"
	// create root path and transactions directory if does not exists
	err = os.MkdirAll(rootPath, 0775)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(rootPath + "transactions/", 0775)
	if err != nil {
		panic(err)
	}
	_, err = os.OpenFile(rootPath + "blockchain.db", os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	db, err = sql.Open("sqlite3", rootPath + "blockchain.db")
	if err != nil {
		panic(err)
	}
	// create blockchain table if does not exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blockchain (height INTEGER PRIMARY KEY, hash TEXT, prev_hash TEXT, merkle_root TEXT, time INTEGER, bits INTEGER, nonce INTEGER)")
	if err != nil {
		fmt.Println(err)
	}
	// get the block the biggest height
lastBlock:
	lastBlockRow := db.QueryRow("SELECT * FROM blockchain ORDER BY height DESC LIMIT 1;")
	LastBlock, err = getBlockFromRow(lastBlockRow)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			bootstrapBlockChain()
			goto lastBlock
		} else {
			panic(err)
		}
	}
	fmt.Printf("hash is %x\n", LastBlock.GetHash())
}

func bootstrapBlockChain() {
	fmt.Println("bootstrapping blockchain...")
	merkleRoot, err := hex.DecodeString("3BA3EDFD7A7B12B27AC72C3E67768F617FC81BC3888A51323A9FB8AA4B1E5E4A")
	if err != nil {
		panic(err)
	}
	scriptSig, err := hex.DecodeString("04FFFF001D0104455468652054696D65732030332F4A616E2F32303039204368616E63656C6C6F72206F6E206272696E6B206F66207365636F6E64206261696C6F757420666F722062616E6B73")
	if err != nil {
		panic(err)
	}
	script, err := hex.DecodeString("4104678AFDB0FE5548271967F1A67130B7105CD6A828E03909A67962E0EA1F61DEB649F6BC3F4CEF38C4F35504E51EC112DE5C384DF7BA0B8D578A4C702B6BF11D5FAC")
	if err != nil {
		panic(err)
	}
	prevTxHash, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000FFFFFFFF")
	if err != nil {
		panic(err)
	}
	genesis := &Block{
		Height: 0,
		Time: 1231006505,
		Bits: 0x1d00ffff,
		Nonce: 2083236893,
	}
	copy(genesis.MerkleRoot[:], merkleRoot)
	genesis.Transactions = make([]*Transaction, 0)
	genesis.Transactions = append(genesis.Transactions, &Transaction{LockTime: 0})
	genesis.Transactions[0].Inputs = make([]*TxIn, 0)
	genesis.Transactions[0].Inputs = append(genesis.Transactions[0].Inputs, &TxIn{})
	copy(genesis.Transactions[0].Inputs[0].PrevTxHash[:], prevTxHash)
	genesis.Transactions[0].Inputs[0].Script = scriptSig
	genesis.Transactions[0].Outputs = make([]*TxOut, 0)
	genesis.Transactions[0].Outputs = append(genesis.Transactions[0].Outputs, &TxOut{Value: 5000000000})
	genesis.Transactions[0].Outputs[0].Script = script
	genesis.Hash = genesis.GetHash()
	addBlockToDb(genesis)
}

func addBlockToDb(block *Block) {
	statement := "INSERT INTO blockchain (height, hash, prev_hash, merkle_root, time, bits, nonce) VALUES (0, $1, $2, $3, $4, $5, $6)"
	hashHex := hex.EncodeToString(block.Hash[:])
	prevHashHex := hex.EncodeToString(block.PrevHash[:])
	merkleRootHex := hex.EncodeToString(block.MerkleRoot[:])
	_, err := db.Exec(statement, hashHex, prevHashHex, merkleRootHex, block.Time, block.Bits, block.Nonce)
	if err != nil {
		panic(err)
	}
	txFile, err := os.Create(rootPath + "transactions/" + strconv.Itoa(block.Height))
	if err != nil {
		panic(err)
	}
	encode := gob.NewEncoder(txFile)
	encode.Encode(block.Transactions)
}

func getBlockFromRow(row *sql.Row) (*Block, error) {
	block := &Block{}
	var hashHex string
	var prevHashHex string
	var merkleRootHex string
	err := row.Scan(&block.Height, &hashHex, &prevHashHex, &merkleRootHex, &block.Time, &block.Bits, &block.Nonce)
	if err != nil {
		return nil, err
	}
	hash, err := hex.DecodeString(hashHex)
	prevHash, err := hex.DecodeString(prevHashHex)
	if err != nil {
		return nil, err
	}
	merkleRoot, err := hex.DecodeString(merkleRootHex)
	if err != nil {
		return nil, err
	}
	copy(block.Hash[:], hash)
	copy(block.PrevHash[:], prevHash)
	copy(block.MerkleRoot[:], merkleRoot)
	txFile, err := os.ReadFile(rootPath + "transactions/" + strconv.Itoa(block.Height))
	if err != nil {
		return nil, err
	}
	decode := gob.NewDecoder(bytes.NewReader(txFile))
	err = decode.Decode(&block.Transactions)
	if err != nil {
		panic(err)
	}
	return block, nil
}

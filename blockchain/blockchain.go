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

	"github.com/davecgh/go-spew/spew"
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
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blockchain (height INTEGER PRIMARY KEY, prev_hash BLOB, merkle_root BLOB, time INTEGER, bits INTEGER, nonce INTEGER)")
	if err != nil {
		fmt.Println(err)
	}
	// get the block the biggest height
	lastBlockRow := db.QueryRow("SELECT * FROM blockchain ORDER BY height DESC LIMIT 1;")
	LastBlock, err = getBlockFromRow(lastBlockRow)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			bootstrapBlockChain()
		} else {
			panic(err)
		}
	}
	spew.Dump(LastBlock)
}

func bootstrapBlockChain() {
	fmt.Println("bootstrapping blockchain...")
	merkleRoot, err := hex.DecodeString("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")
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
		PrevHash: []byte{},
		MerkleRoot: merkleRoot,
		Time: 1231006505,
		Bits: 0x1d00ffff,
		Nonce: 2083236893,
	}
	genesis.Transactions = make([]*Transaction, 0)
	genesis.Transactions = append(genesis.Transactions, &Transaction{LockTime: 0})
	genesis.Transactions[0].Inputs = make([]*TxIn, 0)
	genesis.Transactions[0].Inputs = append(genesis.Transactions[0].Inputs, &TxIn{})
	copy(genesis.Transactions[0].Inputs[0].PrevTxHash[:], prevTxHash)
	genesis.Transactions[0].Inputs[0].Script = scriptSig
	genesis.Transactions[0].Outputs = make([]*TxOut, 0)
	genesis.Transactions[0].Outputs = append(genesis.Transactions[0].Outputs, &TxOut{Value: 5000000000})
	genesis.Transactions[0].Outputs[0].Script = script
	statement := "INSERT INTO blockchain (height, prev_hash, merkle_root, time, bits, nonce) VALUES (0, $1, $2, $3, $4, $5)"
	_, err = db.Exec(statement, genesis.PrevHash, genesis.MerkleRoot, genesis.Time, genesis.Bits, genesis.Nonce)
	if err != nil {
		panic(err)
	}
	txFile, err := os.Create(rootPath + "transactions/" + strconv.Itoa(genesis.Height))
	if err != nil {
		panic(err)
	}
	encode := gob.NewEncoder(txFile)
	encode.Encode(genesis.Transactions)
}

func getBlockFromRow(row *sql.Row) (*Block, error) {
	block := &Block{}
	err := row.Scan(&block.Height, &block.PrevHash, &block.MerkleRoot, &block.Time, &block.Bits, &block.Nonce)
	if err != nil {
		return nil, err
	}
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

package blockchain

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	_, err = os.OpenFile(rootPath + "blockchain.db", os.O_CREATE|os.O_APPEND, 0665)
	if err != nil {
		panic(err)
	}
	db, err = sql.Open("sqlite3", rootPath + "blockchain.db")
	if err != nil {
		panic(err)
	}
	// create blockchain table if does not exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blockchain (height INTEGER PRIMARY KEY, prev_hash BLOB, merkle_root BLOB, time INTEGER, bits INTEGER, nonce INTEGER, transaction_index INTEGER)")
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
			fmt.Println(err)
		}
	}
}

func bootstrapBlockChain() {
	fmt.Println("bootstrapping blockchain...")
	merkleRoot, err := hex.DecodeString("3BA3EDFD7A7B12B27AC72C3E67768F617FC81BC3888A51323A9FB8AA4B1E5E4A")
	if err != nil {
		panic(err)
	}
	genesis := &Block{
		PrevHash: [32]byte{},
	}
	copy(genesis.PrevHash[:], merkleRoot)
}

func getBlockFromRow(row *sql.Row) (*Block, error) {
	var txIndex int
	block := &Block{}
	err := row.Scan(block.Height, block.PrevHash, block.MerkleRoot, block.Time, block.Bits, block.Nonce, txIndex)
	if err != nil {
		return nil, err
	}
	txFile, err := os.ReadFile(rootPath + "transactions/" + strconv.Itoa(txIndex) + ".json")
	err = json.Unmarshal(txFile, &block.Transactions)
	return block, nil
}

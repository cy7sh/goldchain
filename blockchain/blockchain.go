package blockchain

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
//	"github.com/davecgh/go-spew/spew"
)

var db *sql.DB
var LastBlock *Block
var rootPath string // where the blockchain sould be stored

var OrphanBlocks = make([]*Block, 0)

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
	db, err = sql.Open("sqlite3", "file:" + rootPath + "blockchain.db" + "?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	// create blockchain table if does not exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blockchain (height INTEGER PRIMARY KEY, hash TEXT, prev_hash TEXT, merkle_root TEXT, time INTEGER, bits INTEGER, nonce INTEGER, tx INTEGER)")
	if err != nil {
		fmt.Println(err)
	}
	err = RefreshLastBlock()
	if err != nil {
		bootstrapBlockChain()
	}
	go processOrphans()
}

func RefreshLastBlock() error {
	// get the block with biggest height
	lastBlockRow := db.QueryRow("SELECT * FROM blockchain ORDER BY height DESC LIMIT 1;")
	var err error
	LastBlock, err = getBlockFromRow(lastBlockRow)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return err
		} else {
			panic(err)
		}
	}
	return nil
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
	NewBlock(genesis)
}

func NewBlock(block *Block) {
	block.Hash = block.GetHash()
	// this is not genesis
	if LastBlock != nil {
		if !bytes.Equal(LastBlock.Hash[:], block.PrevHash[:]) {
			OrphanBlocks = append(OrphanBlocks, block)
			return
		}
		block.Height = LastBlock.Height + 1
	}
	statement := "INSERT INTO blockchain (height, hash, prev_hash, merkle_root, time, bits, nonce, tx) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	hashHex := hex.EncodeToString(block.Hash[:])
	prevHashHex := hex.EncodeToString(block.PrevHash[:])
	merkleRootHex := hex.EncodeToString(block.MerkleRoot[:])
	var tx int
	if block.Transactions != nil {
		tx = 1
	}
	_, err := db.Exec(statement, block.Height, hashHex, prevHashHex, merkleRootHex, block.Time, block.Bits, block.Nonce, tx)
	if err != nil {
		panic(err)
	}
	// if this is only a header
	if tx == 0{
		RefreshLastBlock()
		return
	}
	txFile, err := os.Create(rootPath + "transactions/" + hashHex)
	if err != nil {
		panic(err)
	}
	encode := gob.NewEncoder(txFile)
	encode.Encode(block.Transactions)
	RefreshLastBlock()
}

func processOrphans() {
	for {
		for i, block := range OrphanBlocks {
			if bytes.Equal(LastBlock.Hash[:], block.PrevHash[:]) {
				NewBlock(block)
				OrphanBlocks = append(OrphanBlocks[:i], OrphanBlocks[i+1:]...)
			}
		}
	}
}

func getBlockFromRow(row *sql.Row) (*Block, error) {
	block := &Block{}
	var hashHex string
	var prevHashHex string
	var merkleRootHex string
	var height int
	var tx int
	err := row.Scan(&height, &hashHex, &prevHashHex, &merkleRootHex, &block.Time, &block.Bits, &block.Nonce, &tx)
	if err != nil {
		return nil, err
	}
	block.Height = height
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
	if tx == 1 {
		txFile, err := os.ReadFile(rootPath + "transactions/" + hashHex)
		if err != nil {
			return nil, err
		}
		decode := gob.NewDecoder(bytes.NewReader(txFile))
		err = decode.Decode(&block.Transactions)
		if err != nil {
			panic(err)
		}
	}
	return block, nil
}

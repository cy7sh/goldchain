package blockchain

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"math/big"

//	"github.com/btcsuite/btcd/txscript"
	_ "github.com/mattn/go-sqlite3"
//	"github.com/davecgh/go-spew/spew"
)

var db *sql.DB
var LastBlock *Block
var FirstHeader *Block
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
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blockchain (height INTEGER PRIMARY KEY, version INTEGER, hash TEXT, prev_hash TEXT, merkle_root TEXT, time INTEGER, bits INTEGER, nonce INTEGER, tx INTEGER)")
	if err != nil {
		fmt.Println(err)
	}
	err = refreshLastBlock()
	if err != nil {
		bootstrapBlockChain()
	}
	refreshFirstHeader()
}

func refreshFirstHeader() {
	// get the first header-only block from the chain
	firstHeaderRow := db.QueryRow("SELECT * FROM blockchain WHERE tx = 1 ORDER BY height LIMIT 1")
	FirstHeader, _ = getBlockFromRow(firstHeaderRow)
}

func refreshLastBlock() error {
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
		Version: 1,
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
	existing, err := getBlockFromHash(block.Hash)
	// this block already exists
	if err == nil {
		// header exists, add transactions
		if existing.Transactions == nil && block.Transactions != nil {
			newTransactions(block)
		}
		return
	}
	// check if PoW is valid
	if !verifyPoW(block) {
		fmt.Println("got invalid PoW")
		return
	}
	// this is not genesis
	if LastBlock != nil {
		if !bytes.Equal(LastBlock.Hash[:], block.PrevHash[:]) {
			// is this block already an orphan
			for _, orphan := range OrphanBlocks {
				if bytes.Equal(block.Hash[:], orphan.Hash[:]) {
					return
				}
			}
			fmt.Println("found an orphan")
			OrphanBlocks = append(OrphanBlocks, block)
			return
		}
		block.Height = LastBlock.Height + 1
	}
	statement := "INSERT INTO blockchain (height, version, hash, prev_hash, merkle_root, time, bits, nonce, tx) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	hashHex := hex.EncodeToString(block.Hash[:])
	prevHashHex := hex.EncodeToString(block.PrevHash[:])
	merkleRootHex := hex.EncodeToString(block.MerkleRoot[:])
	var tx int
	if block.Transactions != nil {
		tx = 1
	}
	_, err = db.Exec(statement, block.Height, block.Version, hashHex, prevHashHex, merkleRootHex, block.Time, block.Bits, block.Nonce, tx)
	if err != nil {
		panic(err)
	}
	// if this is only a header
	if block.Transactions == nil {
		refreshLastBlock()
		refreshFirstHeader()
		return
	}
	newTransactions(block)
	refreshLastBlock()
	refreshFirstHeader()
	processOrphans()
}

func newTransactions(block *Block) error {
	hashHex := hex.EncodeToString(block.Hash[:])
	txFile, err := os.Create(rootPath + "transactions/" + hashHex)
	if err != nil {
		return err
	}
	encode := gob.NewEncoder(txFile)
	encode.Encode(block.Transactions)
	return nil
}

func processOrphans() {
	for i, block := range OrphanBlocks {
		if bytes.Equal(LastBlock.Hash[:], block.PrevHash[:]) {
			fmt.Println("found a parent")
			NewBlock(block)
			OrphanBlocks = append(OrphanBlocks[:i], OrphanBlocks[i+1:]...)
		}
	}
}

func getBlockFromHash(hash [32]byte) (*Block, error) {
	hashHex := hex.EncodeToString(hash[:])
	statement := "SELECT * FROM blockchain WHERE hash = $1"
	return getBlockFromRow(db.QueryRow(statement, hashHex))
}

func getBlockFromHeight(height int) (*Block, error) {
	statement := "SELECT * FROM blockchain WHERE height = $1"
	return getBlockFromRow(db.QueryRow(statement, height))
}

func GetNBlockHashesAfter(start [32]byte, n int) ([][32]byte, error) {
	blocks := make([][32]byte, 0)
	startBlock, err := getBlockFromHash(start)
	if err != nil {
		return nil, err
	}
	startHeight := startBlock.Height + 1
	stopHeight := startHeight + n
	for i := startHeight; i < stopHeight; i++ {
		block, err := getBlockFromHeight(i)
		if err != nil {
			break
		}
		blocks = append(blocks, block.Hash)
	}
	return blocks, nil
}

func getBlockFromRow(row *sql.Row) (*Block, error) {
	block := &Block{}
	var hashHex string
	var prevHashHex string
	var merkleRootHex string
	var height int
	var version int
	var tx int
	err := row.Scan(&height, &version, &hashHex, &prevHashHex, &merkleRootHex, &block.Time, &block.Bits, &block.Nonce, &tx)
	if err != nil {
		return nil, err
	}
	block.Height = height
	block.Version = version
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
			return nil, err
		}
	}
	return block, nil
}

func verifyPoW(block *Block) bool {
	target := compactToBig(uint32(block.Bits))
	hashNum := hashToBig(block.Hash)
	if hashNum.Cmp(target) > 0 {
		return false
	}
	return true
}

// CompactToBig converts a compact representation of a whole number N to an
// unsigned 32-bit number.  The representation is similar to IEEE754 floating
// point numbers.
//
// Like IEEE754 floating point, there are three basic components: the sign,
// the exponent, and the mantissa.  They are broken out as follows:
//
//	* the most significant 8 bits represent the unsigned base 256 exponent
// 	* bit 23 (the 24th bit) represents the sign bit
//	* the least significant 23 bits represent the mantissa
//
//	-------------------------------------------------
//	|   Exponent     |    Sign    |    Mantissa     |
//	-------------------------------------------------
//	| 8 bits [31-24] | 1 bit [23] | 23 bits [22-00] |
//	-------------------------------------------------
//
// The formula to calculate N is:
// 	N = (-1^sign) * mantissa * 256^(exponent-3)
//
// This compact form is only used in bitcoin to encode unsigned 256-bit numbers
// which represent difficulty targets, thus there really is not a need for a
// sign bit, but it is implemented here to stay consistent with bitcoind.
func compactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// HashToBig converts a block.Hash into a big.Int that can be used to
// perform math comparisons.
func hashToBig(hash [32]byte) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	for i := 0; i < 32/2; i++ {
		hash[i], hash[32-1-i] = hash[32-1-i], hash[i]
	}

	return new(big.Int).SetBytes(hash[:])
}

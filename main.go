package main

import (
	"fmt"
	"time"

	"github.com/singurty/goldchain/network"
	"github.com/singurty/goldchain/blockchain"
)

func main() {
	blockchain.Start() // blockchain should be ready before we start the network
	go network.Start()
	for {
		fmt.Printf("total peers: %v\n", len(network.Peers))
		fmt.Printf("orphan blocks: %v\n", len(blockchain.OrphanBlocks))
		time.Sleep(5 * time.Second)
	}
}

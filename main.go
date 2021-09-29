package main

import (
	"fmt"
	"time"

	"github.com/singurty/goldchain/network"
	"github.com/singurty/goldchain/peer"
	"github.com/singurty/goldchain/blockchain"
)

func main() {
	blockchain.Start() // blockchain should be ready before we start the network
	go network.Start()
	for {
		fmt.Printf("total valid peers: %v\n", len(peer.Peers))
		time.Sleep(5 * time.Second)
	}
}

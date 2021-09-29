package main

import (
	"fmt"
	"time"

	"github.com/singurty/goldchain/network"
	"github.com/singurty/goldchain/peer"
)

func main() {
	network.Start()
	for {
		fmt.Printf("total valid peers: %v\n", len(peer.Peers))
		time.Sleep(5 * time.Second)
	}
}

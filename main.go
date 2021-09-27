package main

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/singurty/goldchain/peer"
)

var peers []peer.Peer

func main() {
	nodes := getNodes()
	fmt.Printf("got %v nodes\n", len(nodes))
	for _, node := range nodes {
		go connectToNode(node)
	}
	for {
		fmt.Printf("total valid peers: %v\n", len(peers))
		time.Sleep(5 * time.Second)
	}
}

func getNodes() []net.IP {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("seed.bitcoin.sipa.be"), dns.TypeA)
	in, err := dns.Exchange(m, "8.8.8.8:53")
	if err != nil {
		panic(err)
	}
	nodes := make([]net.IP, 0)
	for _, ans := range in.Answer {
		if t, ok := ans.(*dns.A); ok {
			nodes = append(nodes, t.A)
		}
	}
	return nodes
}

func connectToNode(node net.IP) {
	conn, err := net.Dial("tcp", node.String()+":8333")
	if err != nil {
		return
	}
	peer := peer.Peer{Conn: conn}
	err = peer.Start()
	// is a valid peer
	if err == nil {
		peers = append(peers, peer)
	}
}

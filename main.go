package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/singurty/goldchain/wire"
	"github.com/singurty/goldchain/peer"
)

func main() {
	nodes := getNodes()
	fmt.Printf("got %v nodes\n", len(nodes))
	for _, node := range nodes {
		fmt.Printf("connecting to %v\n", node)
		go connectToNode(node)
	}
	for {

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

func connectToNode(node net.IP) error {
	conn, err := net.Dial("tcp", node.String() + ":8333")
	if err != nil {
		return err
	}
	peer := peer.Peer{Conn: conn}
	go peer.Handler()
	nonceBig, err := rand.Int(rand.Reader, big.NewInt(2^64))
	if err != nil {
		return err
	}
	nonce := nonceBig.Uint64()
	msg := wire.VersionMsg{
		Version: 106, // Bitcoin Core 0.1.6
		Services: 0x00, // Unnamed
		Timestamp: time.Now().Unix(),
		Addr_recv: wire.NetAddr{Services: 0x00, Address: node.To16(), Port: 8333,},
		Addr_from: wire.NetAddr{Services: 0x00, Address: net.ParseIP("::ffff:127.0.0.1"), Port: 0},
		Nonce: nonce,
		User_agent: byte(0x00),
	}
	fmt.Println("sending version message")
	err = msg.Write(conn)
	if err != nil {
		panic(err)
	}
	return nil
}

func peerHandler() {

}

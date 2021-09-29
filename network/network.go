package network

import (
	"net"

	"github.com/miekg/dns"
	"github.com/singurty/goldchain/peer"
)

// computers running bitcoin
var Nodes []net.IP

func Start() {
	Nodes := getNodes()
	for _, node := range Nodes {
		go connectToNode(node)
	}
}

func getNodes() []net.IP {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("seed.bitcoin.sipa.be"), dns.TypeA)
	in, err := dns.Exchange(m, "8.8.8.8:53")
	if err != nil {
		panic(err)
	}
	Nodes := make([]net.IP, 0)
	for _, ans := range in.Answer {
		if t, ok := ans.(*dns.A); ok {
			Nodes = append(Nodes, t.A)
		}
	}
	return Nodes
}

func connectToNode(node net.IP) {
	conn, err := net.Dial("tcp", node.String()+":8333")
	if err != nil {
		return
	}
	peer := peer.Peer{Conn: conn}
	peer.Start()
}

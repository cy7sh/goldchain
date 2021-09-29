package network

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/singurty/goldchain/peer"
)

// computers running bitcoin
var Nodes []*Node
//dns seeds to bootstrap
var seeds []string

type Node struct {
	Address net.IP
	Status int	// node status codes:
				// 0 - not contacted yet
				// 1 - connected, bash a Peer instance
				// 2 - connection was attempted
	Peer *peer.Peer
}

func Start() {
	seeds = []string{"seed.bitcoin.sipa.be", "dnsseed.bluematt.me", "dnsseed.bitcoin.dashjr.org", "seed.bitcoinstats.com", "seed.bitcoin.jonasschnelli.ch", "seed.btc.petertodd.org", "seed.bitcoin.sprovoost.nl", "dnsseed.emzy.de", "seed.bitcoin.wiz.biz"}
	getNodes()
	for {
		for _, node := range Nodes {
			if node.Status == 0 {
				node.Status = 2
				go node.connect()
			}
		}
	}
}

func getNodes() {
	for _, seed := range seeds {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(seed), dns.TypeA)
		c := new(dns.Client)
		c.Net = "tcp"
		c.Timeout = 10 * time.Second
		in, _, err := c.Exchange(m, "8.8.8.8:53")
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, ans := range in.Answer {
			if t, ok := ans.(*dns.A); ok {
				if !doesExist(t.A) {
					node := &Node{Address: t.A}
					Nodes = append(Nodes, node)
				}
			}
		}
	}
}

func doesExist(address net.IP) bool {
	for _, node := range Nodes {
		if node.Address.String() == address.String() {
			return true
		}
	}
	return false
}

func (n *Node) connect() {
	conn, err := net.Dial("tcp", n.Address.String()+":8333")
	if err != nil {
		return
	}
	peer := &peer.Peer{Conn: conn}
	n.Peer = peer
	n.Status = 1
	peer.Start()
}

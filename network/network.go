package network

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"github.com/singurty/goldchain/blockchain"
)

//dns seeds to bootstrap
var seeds []string
// computers that bitcoin
var Nodes []*Node
// peers that we connected to
var Peers []*Peer

var ProtocolVersion = 70013

type Node struct {
	Address net.IP
	Port int
	Status int	// node status codes:
				// 0 - not contacted yet
				// 1 - connected, bash a Peer instance
				// 2 - connection was attempted
	Peer *Peer
}

var maxPeers = 50

var headers = make(chan string)

func Start() {
	seeds = []string{"seed.bitcoin.sipa.be", "dnsseed.bluematt.me", "dnsseed.bitcoin.dashjr.org", "seed.bitcoinstats.com", "seed.bitcoin.jonasschnelli.ch", "seed.btc.petertodd.org", "seed.bitcoin.sprovoost.nl", "dnsseed.emzy.de", "seed.bitcoin.wiz.biz"}
	getNodes()
	for {
		if len(Peers) >= maxPeers {
			break
		}
		for _, node := range Nodes {
			if node.Status == 0 {
				node.Status = 2
				go node.connect()
			}
		}
	}
	go fillBlockchain()
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
					node := &Node{Address: t.A, Port: 8333}
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

func NewNode(address []byte, port int) {
	// might be non-existent
	if port == 0 {
		return
	}
	// check for duplicates
	for _, node := range Nodes {
		if bytes.Equal(node.Address.To16(), address) {
			return
		}
	}
	Nodes = append(Nodes, &Node{Address: address, Port: port})
}

func fillBlockchain() {
	for _, peer := range Peers {
		for {
			fmt.Println("sending getheaders")
			peer.SendGetHeaders(blockchain.LastBlock.Hash, [32]byte{})
			select {
			case msg := <-headers:
				switch msg {
				case "finished":
					continue
				case "best":
					goto next
				}
			case <-time.After(15 * time.Second):
				// not actually best but works
				goto next
			}
		}
next:
	}
}

func (n *Node) connect() {
	conn, err := net.Dial("tcp", n.Address.String() + ":" + strconv.Itoa(n.Port))
	if err != nil {
		return
	}
	peer := &Peer{Conn: conn}
	n.Peer = peer
	n.Status = 1
	peer.Start()
}

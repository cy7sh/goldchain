package main

import (
	"net"

	"github.com/miekg/dns"
)

func main() {
	nodes := getNodes()
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

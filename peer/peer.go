package peer

import (
	"fmt"
	"net"
	"bufio"
)

type Peer struct {
	Conn net.Conn
}

func (p *Peer) Handler() {
	buf := make([]byte, 4096)
	for {
		readLen, err := bufio.NewReader(p.Conn).Read(buf)
		if err != nil {
			fmt.Println("connection closed with", p.Conn.RemoteAddr())
			break
		}
		fmt.Printf("read %v bytes from %v\n", readLen, p.Conn.RemoteAddr().String())
	}
}

package peer

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/singurty/goldchain/wire"
)

var Peers []*Peer

type Peer struct {
	Alive bool
	Conn net.Conn
	version int32
	services uint64
	user_agent string
	start_height int32
	relay bool
}

func (p *Peer) Start()  {
	go p.handler()
	err := p.sendVersion()
	if err != nil {
		fmt.Print(err)
	}
}

func (p *Peer) handler() {
	listen := make(chan string)
	go p.listener(listen)
	for {
		command := <-listen
		switch command {
		case "version":
			// perhaps alive
			p.Alive = true
			Peers = append(Peers, p)
			err := p.sendVerack()
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (p *Peer) listener(c chan string) {
	buf := make([]byte, 4096)
	for {
		_, err := p.Conn.Read(buf)
		if err != nil {
			fmt.Println("connection closed with", p.Conn.RemoteAddr())
			break
		}
		// check if this is a bitcoin message
		magic := binary.LittleEndian.Uint32(buf[:4])
		if !(magic == 0xD9B4BEF9 || magic == 0xDAB5BFFA || magic == 0x0709110B || magic == 0x40CF030A || magic == 0xFEB4BEF9) {
			continue
		}
		command := string(bytes.TrimRight(buf[4:16], "\x00"))
//		fmt.Printf("got %v from %v\n", command, p.Conn.RemoteAddr())
		length := binary.LittleEndian.Uint32(buf[16:20])
		if length == 0 {
//			fmt.Println("empty payload")
			continue
		}
		checksum := buf[20:24]
		var payload []byte
		payload = buf[24:24+length]
		singleHash := sha256.Sum256(payload)
		doubleHash := sha256.Sum256(singleHash[:])
		if !bytes.Equal(checksum, doubleHash[:4]){
//			fmt.Println("corrupt payload")
			continue
		}
		c <- command
		switch command {
		case "version":
//			fmt.Println("parsing version")
			p.version = int32(binary.LittleEndian.Uint32(payload[:4]))
			p.services = binary.LittleEndian.Uint64(payload[4:12])
			user_agent, size, err := wire.ReadVarStr(payload[80:])
			if err != nil {
				fmt.Println(err)
				continue
			}
			p.user_agent = user_agent
			p.start_height = int32(binary.LittleEndian.Uint32(payload[80+size:84+size]))
			// there might not be a relay field
			if uint(length) >= 84 + uint(size) {
				if payload[84+size] == 0x01 {
					p.relay = true
				} else {
					p.relay = false
				}
			}
			fmt.Printf("%v, %x, %v, %v, %v\n", p.version, p.services, p.start_height, p.user_agent, p.relay)
		}
	}
}

func (p *Peer) sendVersion() error {
	nonceBig, err := rand.Int(rand.Reader, big.NewInt(2^64))
	if err != nil {
		return err
	}
	nonce := nonceBig.Uint64()
	msg := wire.VersionMsg{
		Version:    31800,
		Services:   0x00,
		Timestamp:  time.Now().Unix(),
		Addr_recv:  wire.NetAddr{Services: 0x00, Address: net.ParseIP("::ffff:127.0.0.1"), Port: 0},
		Addr_from:  wire.NetAddr{Services: 0x00, Address: net.ParseIP("::ffff:127.0.0.1"), Port: 0},
		Nonce:      nonce,
		User_agent: byte(0x00),
	}
	err = msg.Write(p.Conn)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) sendVerack() error {
	wire.WriteVerackMsg(p.Conn)
	return nil
}

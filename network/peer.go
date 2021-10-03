package network

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net"
	"time"

	"github.com/singurty/goldchain/blockchain"
	"github.com/singurty/goldchain/wire"
)

type Peer struct {
	Alive bool
	Conn net.Conn
	version int32
	services uint64
	user_agent string
	start_height int32
	relay bool
	nonce uint64 // for ping pong
	hc chan string // to signal handler
}

func (p *Peer) Start()  {
	go p.handler()
	err := p.sendVersion()
	if err != nil {
		fmt.Print(err)
		p.hc <- "closed"
	}
}

func (p *Peer) handler() {
	p.hc = make(chan string, 1)
	listen := make(chan string, 5)
	go p.listener(listen)
	for {
		select {
		case command := <-listen:
			switch command {
			case "version":
				// perhaps alive
				p.Alive = true
				Peers = append(Peers, p)
				p.sendVerack()
			case "ping":
				p.sendPong()
			}
		case handle := <-p.hc:
			switch handle {
			// connection closed
			case "closed":
				return
			}
		case <-time.After(10 * time.Minute):
			p.sendPing()
			select {
			case command := <-listen:
				switch command {
				case "pong":
					break
				default:
					listen <- command
				}
			case <-time.After(10 * time.Minute):
				p.Alive = false
			}
		}
	}
}

func (p *Peer) listener(c chan string) {
	buf := make([]byte, 65536)
	bufReader := bufio.NewReader(p.Conn)
	for {
		n, err := bufReader.Read(buf)
		if err != nil {
			p.hc <- "closed"
			return
		}
		// check if this is a bitcoin message
		magic := binary.LittleEndian.Uint32(buf[:4])
		if !(magic == 0xD9B4BEF9 || magic == 0xDAB5BFFA || magic == 0x0709110B || magic == 0x40CF030A || magic == 0xFEB4BEF9) {
			continue
		}
		command := string(bytes.TrimRight(buf[4:16], "\x00"))
		length := int(binary.LittleEndian.Uint32(buf[16:20]))
		payload := make([]byte, length)
		checksum := make([]byte, 4)
		copy(checksum, buf[20:24])
		if length > 0 {
			if length + 24 > n {
				copy(payload, buf[24:n])
				payloadBufIndex := n - 24
				toRead := length - (n - 24)
				for {
					if toRead == 0 {
						break
					}
					n, err := bufReader.Read(payload[payloadBufIndex:])
					if err != nil {
						p.hc <- "closed"
						return
					}
					payloadBufIndex += n
					toRead -= n
				}
			} else {
				copy(payload, buf[24:24+length])
			}
			singleHash := sha256.Sum256(payload)
			doubleHash := sha256.Sum256(singleHash[:])
			if !bytes.Equal(checksum, doubleHash[:4]){
				continue
			}
		}
		switch command {
		case "version":
			c <- "version"
			err := p.parseVersion(payload)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "addr":
			err := p.parseAddr(payload)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "ping":
			p.nonce = binary.LittleEndian.Uint64(payload[:8])
			c <- "ping"
		case "pong":
			nonce := binary.LittleEndian.Uint64(payload[:8])
			if p.nonce == nonce {
				c <- "pong"
			}
		case "headers":
			fmt.Println("yay! got headers")
			err := p.parseHeaders(payload)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	}
}

func (p *Peer) parseVersion(payload []byte) error {
	p.version = int32(binary.LittleEndian.Uint32(payload[:4]))
	p.services = binary.LittleEndian.Uint64(payload[4:12])
	user_agent, size, err := wire.ReadVarStr(payload[80:])
	if err != nil {
		return err
	}
	p.user_agent = user_agent
	p.start_height = int32(binary.LittleEndian.Uint32(payload[80+size:84+size]))
	// there might be a relay field
	if uint(len(payload)) > 84 + uint(size) {
		if payload[84+size] == 0x01 {
			p.relay = true
		} else {
			p.relay = false
		}
	}
	return nil
}

func (p *Peer) parseHeaders(payload []byte) error {
	count, size, err := wire.ReadVarInt(payload)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		prevBlock := payload[size + 4:size + 36]
		merkleRoot := payload[size + 36:size + 68]
		timestamp := int(binary.LittleEndian.Uint32(payload[size + 68:size + 72]))
		bits := int(binary.LittleEndian.Uint32(payload[size + 72:size + 76]))
		nonce := int(binary.LittleEndian.Uint32(payload[size + 76:size + 80]))
		block := &blockchain.Block{
			Time: timestamp,
			Bits: bits,
			Nonce: nonce,
		}
		copy(block.PrevHash[:], prevBlock)
		copy(block.MerkleRoot[:], merkleRoot)
		blockchain.NewBlock(block)
		size += 81
	}
	return nil
}

func (p *Peer) parseAddr(payload []byte) error {
	count, size, err := wire.ReadVarInt(payload)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++{
		offset := size + (i * 30)
		if offset + 30 >= len(payload) {
			break
		}
		address := payload[offset + 12 : offset + 28]
		port := int(binary.BigEndian.Uint16(payload[offset + 28 : offset + 30]))
		Nodes = append(Nodes, &Node{Address: address, Port: port})
	}
	return nil
}

func (p *Peer) sendVersion() error {
	nonceBig, err := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 62))))
	if err != nil {
		return err
	}
	nonce := nonceBig.Uint64()
	msg := wire.VersionMsg{
		Version:    int32(ProtocolVersion),
		Services:   0x00,
		Timestamp:  time.Now().Unix(),
		Addr_recv:  wire.NetAddr{Services: 0x00, Address: net.ParseIP("::ffff:127.0.0.1"), Port: 0},
		Addr_from:  wire.NetAddr{Services: 0x00, Address: net.ParseIP("::ffff:127.0.0.1"), Port: 0},
		Nonce:      nonce,
		User_agent: byte(0x00),
		Relay: false,
	}
	err = msg.Write(p.Conn)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) sendVerack() error {
	return wire.WriteVerackMsg(p.Conn)
}

func (p *Peer) sendPing() error {
	nonceBig, err := rand.Int(rand.Reader, big.NewInt(2^64))
	if err != nil {
		return err
	}
	p.nonce = nonceBig.Uint64()
	return wire.WritePing(p.Conn, p.nonce)
}

func (p *Peer) sendPong() {
	err := wire.WritePong(p.Conn, p.nonce)
	if err != nil {
		p.hc <- "closed"
	}
}

func (p *Peer) sendGetAddr() {
	err := wire.WriteGetaddr(p.Conn)
	if err != nil {
		p.hc <- "closed"
	}
}

func (p *Peer) SendGetHeaders(start [32]byte, end [32]byte) {
	err := wire.WriteGetHeaders(p.Conn, ProtocolVersion, start, end)
	if err != nil {
		p.hc <- "closed"
	}
}

package wire

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type VersionMsg struct {
	Version int32
	Services uint64
	Timestamp int64
	Addr_recv NetAddr
	Addr_from NetAddr
	Nonce uint64
	User_agent string
	Start_height int32
	Relay bool
}

type NetAddr struct {
	Services uint64
	Address net.IP
	Port uint16
}

func (v *VersionMsg) Serialize (w io.Writer) error {
	var scratch [8]byte
	binary.LittleEndian.PutUint32(scratch[0:4], uint32(v.Version))
	w.Write(scratch[0:4])
	return nil
}

func writeElement(w io.Writer, element interface{}) error {
	var scratch [8]byte
	switch e := element.(type) {
	case int32:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, uint32(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
	case int64:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, uint64(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
	case uint32:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, e)
		_, err := w.Write(b)
		if err != nil {
			return err
		}
	case uint64:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, e)
		_, err := w.Write(b)
		if err != nil {
			return err
		}
	case bool:
		b := scratch[0:1]
		if e {
			b[0] = 0x01
		} else {
			b[0] = 0x00
		}
		_, err := w.Write(b)
		if err != nil {
			return err
		}
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
	}
	return nil
}

func writeElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := writeElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}
func writeNetaddress(w io.Writer, addr NetAddr) error {
	err := writeElement(w, addr.Services)
	if err != nil {
		return err
	}
	var ip [16]byte
	copy(ip[:], addr.Address.To16())
	err = writeElement(w, ip)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, addr.Port)
	if err != nil {
		return err
	}
	return nil
}

func writeVarInt(w io.Writer, integer int) error {
	if integer < 0xfd {
		return binary.Write(w, binary.LittleEndian, uint8(integer))
	}
	if integer < 0xffff {
		_, err := w.Write([]byte{0xfd})
		if err != nil {
			return err
		}
		return binary.Write(w, binary.LittleEndian, uint16(integer))
	}
	if integer < 0xffffffff {
		_, err := w.Write([]byte{0xfe})
		if err != nil {
			return err
		}
		return writeElement(w, uint32(integer))
	}
	_, err := w.Write([]byte{0xff})
	if err != nil {
		return err
	}
	return writeElement(w, uint64(integer))
}

func writeVarStr(w io.Writer, element string) error {
	err := writeVarInt(w, len(element))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(element))
	return err
}

func (ver *VersionMsg) Write(w io.Writer) error {
	var b bytes.Buffer
	buf := bufio.NewWriter(&b)
	err := writeElements(buf, ver.Version, ver .Services, ver.Timestamp)
	if err != nil {
		return err
	}
	err = writeNetaddress(buf, ver.Addr_recv)
	if err != nil {
		return err
	}
	err = writeNetaddress(buf, ver.Addr_from)
	if err != nil {
		return err
	}
	err = writeElement(buf, ver.Nonce)
	if err != nil {
		return err
	}
	err = writeVarStr(buf, ver.User_agent)
	if err != nil {
		return err
	}
	err = writeElements(buf, ver.Start_height, ver.Relay)
	if err != nil {
		return err
	}
	buf.Flush()
	bReader := bufio.NewReader(&b)
	msg := make([]byte, bReader.Size())
	msgSize, err := bReader.Read(msg)
	if err != nil {
		return err
	}
	fmt.Printf("version msg is %v bytes\n", msgSize)
	count, err := w.Write(msg)
	fmt.Printf("sent %v bytes\n", count)
	return err
}

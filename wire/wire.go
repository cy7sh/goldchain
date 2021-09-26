package wire

import (
	"encoding/binary"
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
		w.Write(b)
	case int64:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, uint64(e))
		w.Write(b)
	case uint32:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, e)
		w.Write(b)
	case uint64:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, e)
		w.Write(b)
	case bool:
		b := scratch[0:1]
		if e {
			b[0] = 0x01
		} else {
			b[0] = 0x00
		}
	case [16]byte:
		w.Write(e[:])
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
	writeElement(w, addr.Services)
	var ip [16]byte
	copy(ip[:], addr.Address.To16())
	writeElement(w, ip)
	binary.Write(w, binary.BigEndian, addr.Port)
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
	_, err =w.Write([]byte(element))
	return err
}

func (ver *VersionMsg) Write(w io.Writer) error {
	writeElements(w, ver.Version, ver .Services, ver.Timestamp)
	writeNetaddress(w, ver.Addr_recv)
	writeNetaddress(w, ver.Addr_from)
	writeElement(w, ver.Nonce)
	writeVarStr(w, ver.User_agent)
	writeElements(w, ver.Start_height, ver.Relay)
	return nil
}

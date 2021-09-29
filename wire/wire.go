package wire

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
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
	User_agent byte
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
		err := binary.Write(w, binary.BigEndian, e[:])
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

func readVarInt(integer []byte) (int, int, error) {
	if integer[0] < 0xfd {
		return int(integer[0]), 1, nil
	}
	if integer[0] == 0xfd {
		return int(binary.LittleEndian.Uint16(integer[:2])), 2, nil
	}
	if integer[0] == 0xfe {
		return int(binary.LittleEndian.Uint32(integer[:4])), 4, nil
	}
	if integer[0] == 0xff {
		return int(binary.LittleEndian.Uint64(integer[:8])), 8, nil
	}
	return 0, 0, errors.New("invalid var int")
}

func writeVarStr(w io.Writer, element string) error {
	err := writeVarInt(w, len(element))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(element))
	return err
}

func ReadVarStr(str []byte) (string, int, error) {
	length, size, err := readVarInt(str)
	if err != nil {
		return "", 0, err
	}
	return string(str[size:size+length]), size + length, nil
}

func writeMsg(w io.Writer, command string, payload []byte) error {
	var msgBuffer bytes.Buffer
	err := binary.Write(&msgBuffer, binary.LittleEndian, uint32(0xD9B4BEF9))
	if err != nil {
		return err
	}
	_, err = msgBuffer.Write([]byte(command))
	if err != nil {
		return err
	}
	_, err = msgBuffer.Write(make([]byte, 12 - len(command)))
	if err != nil {
		return err
	}
	err = writeElement(&msgBuffer, uint32(len(payload)))
	if err != nil {
		return err
	}
	singleHash := sha256.Sum256(payload)
	doubleHash := sha256.Sum256(singleHash[:])
	_, err = msgBuffer.Write(doubleHash[:4])
	if err != nil {
		return err
	}
	_, err = msgBuffer.Write(payload)
	if err != nil {
		return err
	}
	msg := make([]byte, msgBuffer.Len())
	_, err = msgBuffer.Read(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	return nil
}

func (ver *VersionMsg) Write(w io.Writer) error {
	var payloadBuffer bytes.Buffer
	err := writeElements(&payloadBuffer, ver.Version, ver .Services, ver.Timestamp)
	if err != nil {
		return err
	}
	err = writeNetaddress(&payloadBuffer, ver.Addr_recv)
	if err != nil {
		return err
	}
	err = writeNetaddress(&payloadBuffer, ver.Addr_from)
	if err != nil {
		return err
	}
	err = writeElements(&payloadBuffer, ver.Nonce, ver.User_agent)
	if err != nil {
		return err
	}
	_, err = payloadBuffer.Write([]byte{ver.User_agent})
	if err != nil {
		return err
	}
	err = writeElement(&payloadBuffer, int32(ver.Start_height))
	if err != nil {
		return err
	}
	payload := make([]byte, payloadBuffer.Len())
	_, err = payloadBuffer.Read(payload)
	if err != nil {
		return err
	}
	return writeMsg(w, "version", payload)
}

func WriteVerackMsg(w io.Writer) error {
	return writeMsg(w, "verack", []byte{})
}

func writePingPong(w io.Writer, nonce uint64, command string) error {
	var payloadBuffer bytes.Buffer
	err := writeElement(&payloadBuffer, nonce)
	if err != nil {
		return err
	}
	payload := make([]byte, payloadBuffer.Len())
	_, err = payloadBuffer.Read(payload)
	if err != nil {
		return err
	}
	return writeMsg(w, command, payload)
}

func WritePing(w io.Writer, nonce uint64) error {
	return writePingPong(w, nonce, "ping")
}

func WritePong(w io.Writer, nonce uint64) error {
	return writePingPong(w, nonce, "pong")
}

func WriteGetaddr(w io.Writer) error {
	return writeMsg(w, "getaddr", []byte{})
}

package util

import (
	"errors"
	"io"
)

const SEGMENT_BITS = 0x7F // 0b01111111
const CONTINUE_BIT = 0x80 // 0b10000000

func ReadVarint(r io.Reader) (int32, error) {
	// ux, err := ReadUvarint(r)
	// x := int32(ux >> 1)
	// if ux&1 != 0 {
	// 	x = ^x
	// }

	// return x, err

	var value int32
	var currentByte [1]byte
	bytes_read := 0

	for {
		_, err := r.Read(currentByte[:])
		if err != nil {
			return -1, err
		}

		value |= int32(currentByte[0]&SEGMENT_BITS) << (bytes_read * 7)

		if (currentByte[0] & CONTINUE_BIT) == 0 {
			break
		}

		bytes_read += 1

		if bytes_read >= 5 {
			return -1, errors.New("VarInt is too big")
		}
	}

	return value, nil
}

func WriteVarint(value int32, w io.ByteWriter) (int, error) {
	v := uint32(value)
	for i := 0; ; i++ {
		if (v & ^uint32(SEGMENT_BITS)) == 0 {
			err := w.WriteByte(byte(v))
			return i + 1, err
		}

		err := w.WriteByte(byte((v & SEGMENT_BITS) | CONTINUE_BIT))
		if err != nil {
			return i, err
		}
		v >>= 7
	}
}

func ReadUTF8String(r io.Reader) (string, error) {
	len, err := ReadVarint(r)
	if err != nil {
		return "", err
	}

	buf := make([]byte, len)
	if _, err = io.ReadFull(r, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

// type StatusJsonMapping struct {
// 	Version struct {
// 		Name     string `json:"name"`
// 		Protocol int    `json:"protocol"`
// 	} `json:"version"`

// 	Players struct {
// 		Max    int `json:"max"`
// 		Online int `json:"online"`
// 		Sample []struct {
// 			Name string `json:"name"`
// 			ID   string `json:"id"`
// 		} `json:"sample"`
// 	} `json:"players"`

// 	Description interface{} `json:"description"`
// 	Favicon     string      `json:"favicon,omitempty"`

// 	PreviewsChat       bool `json:"previewsChat,omitempty"`
// 	EnforcesSecureChat bool `json:"enforcesSecureChat,omitempty"`
// }

type Position struct {
	X int32
	Y int32
	Z int32
}

func (p *Position) Serialize() uint64 {
	return (uint64(p.X&0x3FFFFFF) << 38) | (uint64(p.Z&0x3FFFFFF) << 12) | (uint64(p.Y) & 0xFFF)
}

type ConnContext struct {
	Mode uint8
}

func BoolToByte(b bool) byte {
	var r byte
	if b {
		r = 1
	}

	return r
}

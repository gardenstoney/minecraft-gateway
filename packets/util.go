package packets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/google/uuid"
)

type DataType interface {
	WritableField
	ReadableField
}

type WritableField interface {
	Write(buf *bytes.Buffer) error
}

type ReadableField interface {
	Read(r *bytes.Reader) error
}

func BuildPacket(buf *bytes.Buffer, fields ...WritableField) error {
	contentbuf := bytes.NewBuffer(make([]byte, 0))
	for _, f := range fields {
		err := f.Write(contentbuf)
		if err != nil {
			return err
		}
	}

	err := VarInt(contentbuf.Len()).Write(buf)
	if err != nil {
		return err
	}

	_, err = contentbuf.WriteTo(buf)
	return err
}

type Boolean bool

func (df Boolean) Write(buf *bytes.Buffer) error {
	var b byte = 0
	if df {
		b = 1
	}

	return buf.WriteByte(b)
}

func (df *Boolean) Read(r *bytes.Reader) error {
	b, err := r.ReadByte()
	if err != nil {
		return err
	}

	if b == 0 {
		*df = false
	} else if b == 1 {
		*df = true
	} else {
		return errors.New("invalid byte for Boolean field")
	}

	return nil
}

type Byte byte

func (df Byte) Write(buf *bytes.Buffer) error {
	return buf.WriteByte(byte(df))
}

func (df *Byte) Read(r *bytes.Reader) error {
	b, err := r.ReadByte()
	if err != nil {
		return err
	}

	*df = Byte(b)
	return nil
}

type UnsignedShort uint16

func (df UnsignedShort) Write(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, df)
}

func (df *UnsignedShort) Read(r *bytes.Reader) error {
	return binary.Read(r, binary.BigEndian, df)
}

type Int int32

func (df Int) Write(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, df)
}

func (df *Int) Read(r *bytes.Reader) error {
	return binary.Read(r, binary.BigEndian, df)
}

type Long uint64

func (df Long) Write(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, df)
}

func (df *Long) Read(r *bytes.Reader) error {
	return binary.Read(r, binary.BigEndian, df)
}

type String string

func (df String) Write(buf *bytes.Buffer) error {
	len := VarInt(len(df))
	err := len.Write(buf)
	if err != nil {
		return err
	}

	_, err = buf.WriteString(string(df))
	return err
}

func (df *String) Read(r *bytes.Reader) error {
	var len VarInt
	err := len.Read(r)
	if err != nil {
		return err
	}

	buf := make([]byte, len)
	_, err = io.ReadFull(r, buf)
	*df = String(string(buf))

	return err
}

type VarInt int32

const SEGMENT_BITS = 0x7F // 0b01111111
const CONTINUE_BIT = 0x80 // 0b10000000

func ReadVarint(r io.Reader) (int32, error) {
	var value int32
	var currentByte [1]byte
	bytes_read := 0

	for {
		_, err := r.Read(currentByte[:])
		if err != nil {
			return value, err
		}

		value |= int32(currentByte[0]&SEGMENT_BITS) << (bytes_read * 7)

		if (currentByte[0] & CONTINUE_BIT) == 0 {
			break
		}

		bytes_read += 1

		if bytes_read >= 5 {
			return value, errors.New("VarInt is too big")
		}
	}

	return value, nil
}

func (df VarInt) Write(buf *bytes.Buffer) error {
	v := uint32(df)
	for i := 0; ; i++ {
		if (v & ^uint32(SEGMENT_BITS)) == 0 {
			err := buf.WriteByte(byte(v))
			return err
		}

		err := buf.WriteByte(byte((v & SEGMENT_BITS) | CONTINUE_BIT))
		if err != nil {
			return err
		}
		v >>= 7
	}
}

func (df *VarInt) Read(r *bytes.Reader) error {
	value, err := ReadVarint(r)
	*df = VarInt(value)

	return err
}

type Position struct {
	X int32
	Y int32
	Z int32
}

func (df Position) Write(buf *bytes.Buffer) error {
	packed := (uint64(df.X&0x3FFFFFF) << 38) |
		(uint64(df.Z&0x3FFFFFF) << 12) |
		(uint64(df.Y & 0xFFF))

	return binary.Write(buf, binary.BigEndian, packed)
}

func (df *Position) Read(r *bytes.Reader) error {
	var packed uint64
	err := binary.Read(r, binary.BigEndian, &packed)
	if err != nil {
		return err
	}

	df.X = int32((packed >> 38) & 0x3FFFFFF)
	df.Z = int32((packed >> 12) & 0x3FFFFFF)
	df.Y = int32(packed & 0xFFF)

	return nil
}

type UUID uuid.UUID

func (df UUID) Write(buf *bytes.Buffer) error {
	_, err := buf.Write(df[:])
	return err
}

func (df *UUID) Read(r *bytes.Reader) error {
	_, err := io.ReadFull(r, df[:])
	return err
}

func (df UUID) String() string {
	return uuid.UUID(df).String()
}

// only pointers to DataType objects can be stored,
// leading to data scattered on heap
// could be optimized???
type PrefixedArray[T DataType] struct {
	Items []T
}

func (df PrefixedArray[T]) Write(buf *bytes.Buffer) error {
	if err := VarInt(len(df.Items)).Write(buf); err != nil {
		return err
	}

	for _, item := range df.Items {
		if err := item.Write(buf); err != nil { // crash if df.Item is nil
			return err
		}
	}

	return nil
}

func (df *PrefixedArray[T]) Read(r *bytes.Reader) error {
	var len VarInt
	if err := len.Read(r); err != nil {
		return err
	}

	df.Items = make([]T, len)

	for i := 0; i < int(len); i++ {
		var item T
		if err := item.Read(r); err != nil { // TODO: crash if df.Item is nil
			return err
		}
		df.Items[i] = item
	}

	return nil
}

// PrefixedArray of Byte optimized
type ByteArray []byte

func (df ByteArray) Write(buf *bytes.Buffer) error {
	if err := VarInt(len(df)).Write(buf); err != nil {
		return err
	}

	_, err := buf.Write(df)

	return err
}

func (df *ByteArray) Read(r *bytes.Reader) error {
	var len VarInt
	if err := len.Read(r); err != nil {
		return err
	}

	*df = make([]byte, len)

	_, err := io.ReadFull(r, *df)
	return err
}

type RawBytes []byte

func (df RawBytes) Write(buf *bytes.Buffer) error {
	_, err := buf.Write(df)

	return err
}

// TODO: nah just make it check if it's nil, also T should be always pointer. make it explicit.
// all other DataTypes shouldn't be able to call Read if it's nil.
// only one's in the Optional DataType can be.
type Optional[T DataType] struct {
	Exists Boolean
	Item   T
}

func (df Optional[T]) Write(buf *bytes.Buffer) error {
	if err := df.Exists.Write(buf); err != nil {
		return err
	}

	if df.Exists {
		if err := df.Item.Write(buf); err != nil { // crash if df.Item is nil
			return err
		}
	}

	return nil
}

func (df *Optional[T]) Read(r *bytes.Reader) error {
	if err := df.Exists.Read(r); err != nil {
		return err
	}

	if df.Exists {
		if err := df.Item.Read(r); err != nil { // TODO: crash if df.Item is nil
			return err
		}
	}
	return nil
}

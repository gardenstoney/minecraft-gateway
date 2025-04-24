package packets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/google/uuid"
)

// DataField wraps types that implement both WritableField and ReadableField.
//
// It represents a field in a packet that can be serialized and deserialized.
type DataField interface {
	WritableField
	ReadableField
}

// A WritableField represents a field in a packet that can be serialized.
type WritableField interface {
	Write(buf *bytes.Buffer) error
}

// A ReadableField represents a field in a packet that can be deserialized.
type ReadableField interface {
	Read(r *bytes.Reader) error
}

// BuildPacket takes a series of WritableFields composing a packet,
// builds and serializes a packet to buf
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

// DataField Boolean represents bool field in a packet
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

// DataField Byte represents byte field in a packet
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

// DataField UnsignedShort represents uint16 field in a packet
type UnsignedShort uint16

func (df UnsignedShort) Write(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, df)
}

func (df *UnsignedShort) Read(r *bytes.Reader) error {
	return binary.Read(r, binary.BigEndian, df)
}

// DataField Int represents int32 field in a packet
type Int int32

func (df Int) Write(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, df)
}

func (df *Int) Read(r *bytes.Reader) error {
	return binary.Read(r, binary.BigEndian, df)
}

// DataField Long represents int64 field in a packet
type Long int64

func (df Long) Write(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, df)
}

func (df *Long) Read(r *bytes.Reader) error {
	return binary.Read(r, binary.BigEndian, df)
}

// DataField String represents string field in a packet
//
// Serialized String is prefixed with a VarInt of its length
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

// DataField VarInt represents variable-length int32 field in a packet
//
// The most significant bit of each byte indicates whether more bytes follow:
// 1 if more bytes follow, 0 if not.
//
// Serialized VarInt can be from 1 byte to 5 bytes long
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

// DataField Position represents position field in a packet
//
// Serialized form is composed of X, Z which are 26 bits each, and 12 bits of Y.
// Thus, unintended content can be written when the values are out of range
type Position struct {
	X int32
	Y int16
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
	df.Y = int16(packed & 0xFFF)

	return nil
}

// DataField UUID represents UUID field in a packet
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

// DataField PrefixedArray[T] represents length-prefixed array of field T
//
// # Caution
//
// Make sure to initialize elements of PrefixedArray properly
// since T would typically be a pointer, whose zero value is nil.
// Use MakePrefixedArray when creating a non-empty array.
//
// # Note
//
// T is typically a pointer type, since Read() requires pointer receivers.
// This means the slice will store pointers, leading to data scattered on heap.
// Consider optimizing if performance is critical.
type PrefixedArray[T DataField] []T

func MakePrefixedArray[T DataField](len int, cap int, newElem func() T) PrefixedArray[T] {
	df := make(PrefixedArray[T], len, cap)
	for i := 0; i < len; i++ {
		df[i] = newElem()
	}

	return df
}

func (df PrefixedArray[T]) Write(buf *bytes.Buffer) error {
	if err := VarInt(len(df)).Write(buf); err != nil {
		return err
	}

	for _, item := range df {
		if err := item.Write(buf); err != nil { // crash if item is nil
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

	for i := 0; i < int(len); i++ {
		if err := (*df)[i].Read(r); err != nil { // crash if df[i] is nil
			return err
		}
	}

	return nil
}

// Optimized PrefixedArray[Byte]
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

// WritableDataField RawBytes is serialized directly into its byte contents.
//
// Use it to flexibly serialize a field and integrate it with BuildPacket
type RawBytes []byte

func (df RawBytes) Write(buf *bytes.Buffer) error {
	_, err := buf.Write(df)

	return err
}

// DataField Optional[T] represents Optional field in a packet
//
// Serialized Optional[T] is prefixed with Boolean of whether the value exists.
// If so, the value T is followed.
//
// # Caution
//
// Make sure to initialize Item properly since T is typically a pointer.
// A nil Optional Item with Exists marked as true could crash the program.
type Optional[T DataField] struct {
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
		if err := df.Item.Read(r); err != nil { // crash if df.Item is nil
			return err
		}
	}
	return nil
}

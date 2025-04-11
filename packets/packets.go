//go:generate go run ../codegen/generate_read.go -- .
package packets

import (
	"bytes"
	"io"
)

type ServerboundPacket interface {
	Read(r *bytes.Reader) error
}

type ClientboundPacket interface {
	Write(buf *bytes.Buffer) error
}

func ReadPacket(r io.Reader) (packetID int32, bufreader *bytes.Reader, err error) {
	payloadLength, err := ReadVarint(r)

	if err != nil {
		return -1, nil, err
	}

	// Read the payload
	payload := make([]byte, payloadLength)
	_, err = io.ReadFull(r, payload)
	if err != nil {
		return -1, nil, err
	}

	bufreader = bytes.NewReader(payload)

	packetID, err = ReadVarint(bufreader)
	if err != nil {
		return -1, nil, err
	}

	return packetID, bufreader, nil
}

// @gen:read
type HandshakePacket struct {
	ProtocolVersion VarInt
	ServerAddr      String
	ServerPort      UnsignedShort
	RequestType     VarInt
}

func (p HandshakePacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(0),
		p.ProtocolVersion,
		p.ServerAddr,
		p.ServerPort,
		p.RequestType,
	)
}

package packets

import (
	"bytes"
)

var LoginPacketRegistry = map[int32]func() ServerboundPacket{
	0: func() ServerboundPacket { return &LoginStartPacket{} },
	1: func() ServerboundPacket { return &EncryptionRespPacket{} },
	3: func() ServerboundPacket { return &LoginAckPacket{} },
}

// @gen:read
type LoginStartPacket struct {
	Name       String
	PlayerUUID UUID
}

// @gen:read
type EncryptionRespPacket struct {
	SharedSecret ByteArray
	VerifyToken  ByteArray
}

// @gen:read
type LoginAckPacket struct{}

type LoginDisconnectPacket struct {
	Reason String
}

func (p LoginDisconnectPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(0),
		p.Reason,
	)
}

type EncryptionReqPacket struct {
	ServerID    String
	PublicKey   ByteArray
	VerifyToken ByteArray
	ShouldAuth  Boolean
}

func (p EncryptionReqPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(1),
		p.ServerID,
		p.PublicKey,
		p.VerifyToken,
		p.ShouldAuth,
	)
}

type LoginSuccessPacket struct {
	UUID              UUID
	Username          String
	Property          PrefixedArray[*Byte] // TODO: needs Property struct implemented
	StrictErrHandling Boolean
}

func (p LoginSuccessPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(2),
		p.UUID,
		p.Username,
		p.Property,
		p.StrictErrHandling,
	)
}

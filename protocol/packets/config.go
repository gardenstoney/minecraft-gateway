package packets

import (
	"bytes"
)

var ConfigPacketRegistry = map[int32]func() ServerboundPacket{
	0: func() ServerboundPacket { return &ClientInfoPacket{} },
	2: func() ServerboundPacket { return &PluginMsgPacket{} },
	3: func() ServerboundPacket { return &AckFinishConfigPacket{} },
	4: func() ServerboundPacket { return &ConfigKeepAlivePacket{} },
}

// @gen:read
type ClientInfoPacket struct {
	Locale              String
	ViewDistance        Byte
	ChatMode            VarInt
	ChatColors          Boolean
	DisplayedSkinParts  Byte // unsigned byte?
	MainHand            VarInt
	EnableTextFiltering Boolean
	AllowServerListings Boolean
}

type PluginMsgPacket struct {
	Channel String
	Data    []byte
}

func (p *PluginMsgPacket) Read(r *bytes.Reader) error {
	if err := p.Channel.Read(r); err != nil {
		return err
	}

	// needs refactoring
	// io.ReadAll()  // all the remaining parts of the packet, whose length can't be determined
	// won't use this packet anyway so...

	return nil
}

// @gen:read
type AckFinishConfigPacket struct{}

// TODO: need NBT encoding and decoding
type ConfigDisconnectPacket struct {
	Reason String
}

func (p ConfigDisconnectPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(2),
		Byte(8),                      // TAG_String NBT id
		UnsignedShort(len(p.Reason)), // NBT payload length
		RawBytes([]byte(p.Reason)),
	)
}

type FinishConfigPacket struct{}

func (p FinishConfigPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(3),
	)
}

// @gen:read
type ConfigKeepAlivePacket struct {
	KeepAliveID Long
}

func (p ConfigKeepAlivePacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(4),
		p.KeepAliveID,
	)
}

type ConfigTransferPacket struct {
	Host String
	Port VarInt
}

func (p ConfigTransferPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(0x0b),
		p.Host,
		p.Port,
	)
}

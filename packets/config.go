package packets

import (
	"bytes"
	"fmt"
	"mcmockserver/util"
	"net"
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

// stuck here :( client complains that registries are missing: ResourceKey[minecraft:root / minecraft:dimension_type]
func (p *AckFinishConfigPacket) Handle(conn net.Conn, context *util.ConnContext) error {
	fmt.Println("Acknowledged config finish, switching to ingame mode")
	context.Mode = 5

	// buf := []byte{0x6c, 0x2b, 0x0, 0x0, 0x0, 0x56, 0x0, 0x3, 0x13, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6f, 0x76, 0x65, 0x72, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x14, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x74, 0x68, 0x65, 0x5f, 0x6e, 0x65, 0x74, 0x68, 0x65, 0x72, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x74, 0x68, 0x65, 0x5f, 0x65, 0x6e, 0x64, 0x14, 0xa, 0xa, 0x0, 0x1, 0x0, 0x0, 0x13, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6f, 0x76, 0x65, 0x72, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0xa5, 0xf2, 0xc7, 0x21, 0x34, 0xb7, 0xdd, 0xda, 0x0, 0xff, 0x0, 0x0, 0x0, 0x0, 0x0}
	// _, err := conn.Write(buf)

	// this thing nasty
	var a, b, c = String("minecraft:overworld"), String("minecraft:the_nether"), String("minecraft:the_end")

	buf := bytes.NewBuffer(make([]byte, 0, 18))
	err := LoginPacket{
		EntityId:            0x56,
		IsHardcore:          false,
		DimensionNames:      PrefixedArray[*String]{[]*String{&a, &b, &c}},
		MaxPlayers:          20,
		ViewDist:            10,
		SimulationDist:      10,
		ReduceDebugInfo:     false,
		EnableRespawnScreen: false,
		LimitCrafting:       false,
		DimensionType:       0,
		DimensionName:       "minecraft:overworld",
		HashedSeed:          11957838906054401498,
		GameMode:            0,
		PrevGameMode:        0xff,
		IsDebug:             false,
		IsFlat:              false,
	}.Write(buf)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(conn)
	return err
}

// FIX: Reason is not just json formated string?
type ConfigDisconnectPacket struct {
	Reason String
}

func (p ConfigDisconnectPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(2),
		p.Reason,
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

func (p *ConfigKeepAlivePacket) Handle(conn net.Conn, context *util.ConnContext) error {
	return nil
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

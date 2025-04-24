package packets

import (
	"bytes"
)

var IngamePacketRegistry = map[int32]func() ServerboundPacket{
	// 1: func() Packet { return &PlayerJoinPacket{} },
}

type LoginPacket struct {
	EntityId            Int
	IsHardcore          Boolean
	DimensionNames      PrefixedArray[*String]
	MaxPlayers          VarInt
	ViewDist            VarInt
	SimulationDist      VarInt
	ReduceDebugInfo     Boolean
	EnableRespawnScreen Boolean
	LimitCrafting       Boolean
	DimensionType       VarInt
	DimensionName       String
	HashedSeed          Long
	GameMode            Byte
	PrevGameMode        Byte
	IsDebug             Boolean
	IsFlat              Boolean
	HasDeathLocation    Boolean
	DeathDimensionName  Optional[*String]
	DeathLocation       Optional[*Position]
	PortalCooldown      VarInt
	EnforcesSecureChat  Boolean
}

func (p LoginPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(0x2b),
		p.EntityId,
		p.IsHardcore,
		p.DimensionNames,
		p.MaxPlayers,
		p.ViewDist,
		p.SimulationDist,
		p.ReduceDebugInfo,
		p.EnableRespawnScreen,
		p.LimitCrafting,
		p.DimensionType,
		p.DimensionName,
		p.HashedSeed,
		p.GameMode,
		p.PrevGameMode,
		p.IsDebug,
		p.IsFlat,
		p.HasDeathLocation,
		p.DeathDimensionName,
		p.DeathLocation,
		p.PortalCooldown,
		p.EnforcesSecureChat,
	)
}

// func (p *LoginPacket) Serialize() (*bytes.Buffer, error) {
// 	buf := bytes.NewBuffer(make([]byte, 0, 18))
// 	buf.WriteByte(0x2B) // Packet ID
// 	binary.Write(buf, binary.BigEndian, p.EntityId)
// 	buf.WriteByte(util.BoolToByte(p.IsHardcore))
// 	util.WriteVarint(int32(len(p.DimensionNames)), buf)

// 	for _, str := range p.DimensionNames {
// 		util.WriteVarint(int32(len(str)), buf)
// 		buf.WriteString(str)
// 	}

// 	util.WriteVarint(p.MaxPlayers, buf)
// 	util.WriteVarint(p.ViewDist, buf)
// 	util.WriteVarint(p.SimulationDist, buf)
// 	buf.WriteByte(util.BoolToByte(p.ReduceDebugInfo))
// 	buf.WriteByte(util.BoolToByte(p.EnableRespawnScreen))
// 	buf.WriteByte(util.BoolToByte(p.LimitCrafting))
// 	util.WriteVarint(p.DimensionType, buf)

// 	util.WriteVarint(int32(len(p.DimensionName)), buf)
// 	buf.WriteString(p.DimensionName)

// 	binary.Write(buf, binary.BigEndian, p.HashedSeed)
// 	buf.WriteByte(p.GameMode)
// 	buf.WriteByte(byte(p.PrevGameMode))
// 	buf.WriteByte(util.BoolToByte(p.IsDebug))
// 	buf.WriteByte(util.BoolToByte(p.IsFlat))

// 	buf.WriteByte(util.BoolToByte(p.HasDeathLocation))
// 	if p.HasDeathLocation {
// 		util.WriteVarint(int32(len(p.DeathDimensionName)), buf)
// 		buf.WriteString(p.DeathDimensionName)
// 		binary.Write(buf, binary.BigEndian, p.DeathLocation.Serialize()) // confusing with endianness
// 	}

// 	util.WriteVarint(p.PortalCooldown, buf)
// 	buf.WriteByte(util.BoolToByte(p.EnforcesSecureChat))

// 	return buf, nil
// }

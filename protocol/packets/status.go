package packets

import "bytes"

var StatusPacketRegistry = map[int32]func() ServerboundPacket{
	0: func() ServerboundPacket { return &StatusReqPacket{} },
	1: func() ServerboundPacket { return &PingReqPacket{} },
}

// @gen:read
type StatusReqPacket struct{}

func (p StatusReqPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(0),
	)
}

// @gen:read
type PingReqPacket struct {
	Timestamp Long
}

// @gen:read
type StatusRespPacket struct {
	Response String
}

func (p StatusRespPacket) Write(buf *bytes.Buffer) error {
	return BuildPacket(
		buf,
		VarInt(0),
		p.Response,
	)
}

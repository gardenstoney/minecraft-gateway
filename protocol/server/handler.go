package server

import (
	"bytes"
	"log/slog"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
)

type PacketHandler func(session *Session, packet packets.ServerboundPacket)

const serverlistping = "{\"version\": {\"name\": \"1.21.1\",\"protocol\": 767},\"players\": {\"max\": 100,\"online\": 0,\"sample\": []},\"description\": {\"text\": \"Hello, world!\"},\"favicon\": \"data:image/png;base64,<data>\",\"enforcesSecureChat\": false}"

func DefaultStatusReqPacketHandler(session *Session) error {
	slog.Info("Status request", "session", session.Transport.String())

	// Status response
	buf := bytes.NewBuffer(make([]byte, 0))
	err := packets.StatusRespPacket{
		Response: serverlistping,
	}.Write(buf)
	if err != nil {
		slog.Error("Failed to build status response", "error", err, "session", session.Transport.String())
		return err
	}

	session.Transport.Write(buf.Bytes())

	slog.Debug("Sent status response", "session", session.Transport.String())
	return nil
}

func DefaultPingReqPacketHandler(session *Session, p *packets.PingReqPacket) error {
	slog.Info("Ping request", "session", session.Transport.String())

	// Pong response
	buf := bytes.NewBuffer(make([]byte, 0, 10))
	err := packets.BuildPacket(
		buf,
		packets.VarInt(1),
		p.Timestamp,
	)
	if err != nil {
		slog.Error("Failed to build pong response", "error", err, "session", session.Transport.String())
		return err
	}

	session.Transport.Write(buf.Bytes())

	slog.Debug("Sent pong response, closing connection", "session", session.Transport.String())
	session.Shutdown()
	return nil
}

func DefaultLoginStartPacketHandler(session *Session, p *packets.LoginStartPacket) error {
	slog.Info("Login request", "session", session.Transport.String())

	buf := bytes.NewBuffer(make([]byte, 0, 18))
	err := packets.LoginSuccessPacket{
		UUID:              p.PlayerUUID,
		Username:          p.Name,
		Property:          packets.PrefixedArray[*packets.Byte]{},
		StrictErrHandling: packets.Boolean(true),
	}.Write(buf)
	if err != nil {
		return err
	}

	session.Transport.Write(buf.Bytes())
	slog.Debug("Sent LoginSuccess", "session", session.Transport.String())

	return nil
}

func DefaultLoginAckPacketHandler(session *Session) error {
	slog.Debug("Login Acknowledged, switching to configuration mode", "session", session.Transport.String())
	session.Mode = Config
	return nil
}

// stuck here :( client complains about registries when sending one of these registries and finish config
// func (p *ClientInfoPacket) Handle(conn net.Conn, context *util.ConnContext) error {
// paintingvariantReg := []byte{0xc2, 0x7, 0x7, 0x1a, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x61, 0x69, 0x6e, 0x74, 0x69, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e, 0x74, 0x32, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x61, 0x6c, 0x62, 0x61, 0x6e, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x61, 0x7a, 0x74, 0x65, 0x63, 0x0, 0x10, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x61, 0x7a, 0x74, 0x65, 0x63, 0x32, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x61, 0x63, 0x6b, 0x79, 0x61, 0x72, 0x64, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x61, 0x72, 0x6f, 0x71, 0x75, 0x65, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x6f, 0x6d, 0x62, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x6f, 0x75, 0x71, 0x75, 0x65, 0x74, 0x0, 0x17, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x75, 0x72, 0x6e, 0x69, 0x6e, 0x67, 0x5f, 0x73, 0x6b, 0x75, 0x6c, 0x6c, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x75, 0x73, 0x74, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x63, 0x61, 0x76, 0x65, 0x62, 0x69, 0x72, 0x64, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x69, 0x6e, 0x67, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x63, 0x6f, 0x74, 0x61, 0x6e, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x63, 0x6f, 0x75, 0x72, 0x62, 0x65, 0x74, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x63, 0x72, 0x65, 0x65, 0x62, 0x65, 0x74, 0x0, 0x15, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x64, 0x6f, 0x6e, 0x6b, 0x65, 0x79, 0x5f, 0x6b, 0x6f, 0x6e, 0x67, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x65, 0x61, 0x72, 0x74, 0x68, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x65, 0x6e, 0x64, 0x62, 0x6f, 0x73, 0x73, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x66, 0x65, 0x72, 0x6e, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x66, 0x69, 0x67, 0x68, 0x74, 0x65, 0x72, 0x73, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x66, 0x69, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x66, 0x69, 0x72, 0x65, 0x0, 0x10, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x67, 0x72, 0x61, 0x68, 0x61, 0x6d, 0x0, 0x10, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x68, 0x75, 0x6d, 0x62, 0x6c, 0x65, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6b, 0x65, 0x62, 0x61, 0x62, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6c, 0x6f, 0x77, 0x6d, 0x69, 0x73, 0x74, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6d, 0x61, 0x74, 0x63, 0x68, 0x0, 0x14, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6d, 0x65, 0x64, 0x69, 0x74, 0x61, 0x74, 0x69, 0x76, 0x65, 0x0, 0xd, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6f, 0x72, 0x62, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6f, 0x77, 0x6c, 0x65, 0x6d, 0x6f, 0x6e, 0x73, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x61, 0x73, 0x73, 0x61, 0x67, 0x65, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x69, 0x67, 0x73, 0x63, 0x65, 0x6e, 0x65, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x6f, 0x6e, 0x64, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x6f, 0x6f, 0x6c, 0x0, 0x16, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x72, 0x61, 0x69, 0x72, 0x69, 0x65, 0x5f, 0x72, 0x69, 0x64, 0x65, 0x0, 0xd, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x65, 0x61, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x6b, 0x65, 0x6c, 0x65, 0x74, 0x6f, 0x6e, 0x0, 0x19, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x6b, 0x75, 0x6c, 0x6c, 0x5f, 0x61, 0x6e, 0x64, 0x5f, 0x72, 0x6f, 0x73, 0x65, 0x73, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x74, 0x61, 0x67, 0x65, 0x0, 0x14, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x75, 0x6e, 0x66, 0x6c, 0x6f, 0x77, 0x65, 0x72, 0x73, 0x0, 0x10, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x75, 0x6e, 0x73, 0x65, 0x74, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x74, 0x69, 0x64, 0x65, 0x73, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x75, 0x6e, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x64, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x76, 0x6f, 0x69, 0x64, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x61, 0x6e, 0x64, 0x65, 0x72, 0x65, 0x72, 0x0, 0x13, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x61, 0x73, 0x74, 0x65, 0x6c, 0x61, 0x6e, 0x64, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x61, 0x74, 0x65, 0x72, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x69, 0x6e, 0x64, 0x0, 0x10, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x69, 0x74, 0x68, 0x65, 0x72, 0x0}
// wolfvariantReg := []byte{0xb8, 0x1, 0x7, 0x16, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x6f, 0x6c, 0x66, 0x5f, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e, 0x74, 0x9, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x61, 0x73, 0x68, 0x65, 0x6e, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x62, 0x6c, 0x61, 0x63, 0x6b, 0x0, 0x12, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x63, 0x68, 0x65, 0x73, 0x74, 0x6e, 0x75, 0x74, 0x0, 0xe, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x70, 0x61, 0x6c, 0x65, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x72, 0x75, 0x73, 0x74, 0x79, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x6e, 0x6f, 0x77, 0x79, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x70, 0x6f, 0x74, 0x74, 0x65, 0x64, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x73, 0x74, 0x72, 0x69, 0x70, 0x65, 0x64, 0x0, 0xf, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x77, 0x6f, 0x6f, 0x64, 0x73, 0x0}
// dimensiontypeReg := []byte{0x74, 0x7, 0x18, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x64, 0x69, 0x6d, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x4, 0x13, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6f, 0x76, 0x65, 0x72, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x0, 0x19, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x6f, 0x76, 0x65, 0x72, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x5f, 0x63, 0x61, 0x76, 0x65, 0x73, 0x0, 0x11, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x74, 0x68, 0x65, 0x5f, 0x65, 0x6e, 0x64, 0x0, 0x14, 0x6d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x3a, 0x74, 0x68, 0x65, 0x5f, 0x6e, 0x65, 0x74, 0x68, 0x65, 0x72, 0x0}

// conn.Write(paintingvariantReg)
// conn.Write(wolfvariantReg)
// conn.Write(dimensiontypeReg)

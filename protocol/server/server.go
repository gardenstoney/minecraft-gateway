package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
)

type ConnectionMode byte

const (
	_ ConnectionMode = iota
	Status
	Login
	Transfer
	Config
	Play
)

type Session struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	Transport Transporter
	Mode      ConnectionMode
	Wg        sync.WaitGroup
	once      sync.Once
}

func (s *Session) Shutdown() {
	s.once.Do(func() {
		s.Cancel()
		// wait for the goroutines to finish
		s.Wg.Wait()

		slog.Info("Closing session", "session", s.Transport.String())
		s.Transport.Close()
	})
}

func NewSession(transport Transporter, parentCtx context.Context) *Session {
	ctx, cancel := context.WithCancel(parentCtx)

	session := Session{
		Ctx:       ctx,
		Cancel:    cancel,
		Transport: transport,
	}

	return &session
}

func HandleSession(session *Session, handler PacketHandler) {
	defer session.Shutdown()

	slog.Info("New session", "session", session.Transport.String())

	// Handshake
	payload, err := session.Transport.Read(session.Ctx)
	if err != nil {
		slog.Error("Failed to read handshake packet", "error", err, "session", session.Transport.String())
		return
	}

	handshakeReader := bytes.NewReader(payload)
	packetID, err := packets.ReadVarint(handshakeReader)
	if err != nil {
		slog.Error("Failed to read handshake packet id", "error", err, "session", session.Transport.String())
		return
	}

	if packetID != 0 {
		slog.Error("Invalid handshake packet type", "error", err, "session", session.Transport.String())
		return
	}

	handshake := packets.HandshakePacket{}
	handshake.Read(handshakeReader)

	slog.Debug(fmt.Sprint("Received handshake", handshake), "session", session.Transport.String())

	session.Mode = ConnectionMode(handshake.RequestType) // a bit risky?? checking if it's valid first is a good idea

	var packetRegistry *map[int32]func() packets.ServerboundPacket

	for {
		// Return if the context was canceled
		select {
		case <-session.Ctx.Done():
			return
		default:
		}

		switch session.Mode {
		case Status:
			packetRegistry = &packets.StatusPacketRegistry
		case Login:
			packetRegistry = &packets.LoginPacketRegistry
		case Transfer:
			packetRegistry = &packets.LoginPacketRegistry // assuming transfer process would be the same
		case Config:
			packetRegistry = &packets.ConfigPacketRegistry
		case Play:
			packetRegistry = &packets.IngamePacketRegistry
		}

		payload, err := session.Transport.Read(session.Ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.Error("Read error", "error", err, "session", session.Transport.String())
			}
			return
		}

		reader := bytes.NewReader(payload)
		packetID, err := packets.ReadVarint(reader)
		if err != nil {
			return
		}

		packetFactory, exists := (*packetRegistry)[packetID]

		if !exists {
			slog.Error("Unknown packet ID", "packetID", packetID, "session", session.Transport.String())
			return
		}

		packet := packetFactory()

		if err := packet.Read(reader); err != nil {
			slog.Error("Failed to read packet content", "error", err, "session", session.Transport.String())
			return
		}

		handler(session, packet)
	}
}

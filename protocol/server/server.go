package server

import (
	"bytes"
	"context"
	"fmt"
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

		fmt.Println("Closing Session:", s.Transport)
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

	// Handshake
	payload, err := session.Transport.Read(session.Ctx)
	if err != nil {
		fmt.Println("Failed to read handshake packet:", err)
		return
	}

	handshakeReader := bytes.NewReader(payload)
	packetID, err := packets.ReadVarint(handshakeReader)
	if err != nil {
		fmt.Println("Failed to read handshake packet id:", err)
		return
	}

	if packetID != 0 {
		fmt.Println("Error handshaking: Unexpected packet type")
		return
	}

	handshake := packets.HandshakePacket{}
	handshake.Read(handshakeReader)

	fmt.Println("Received handshake", handshake)

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
			fmt.Println("read packet loop error:", err)
			return
		}

		reader := bytes.NewReader(payload)
		packetID, err := packets.ReadVarint(reader)
		if err != nil {
			return
		}

		fmt.Printf("Received packet (ID: %d): %x\n", packetID, payload)

		packetFactory, exists := (*packetRegistry)[packetID]

		if !exists {
			fmt.Println("Unknown packet ID:", packetID)
			return
		}

		packet := packetFactory()

		if err := packet.Read(reader); err != nil {
			fmt.Println("Failed to read packet content:", err)
			return
		}

		handler(session, packet)
	}
}

package server

import (
	"fmt"
	"mcmockserver/packets"
	"net"
)

type ConnectionMode byte

const (
	Disconnect ConnectionMode = iota
	Status
	Login
	Transfer
	Config
	Play
)

// Handle incoming client connections
func HandleClient(conn net.Conn, handler PacketHandler) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	// Handshake
	packetID, bufreader, _, err := packets.ReadPacket(conn)
	if err != nil {
		fmt.Println("Failed to read handshake packet:", err)
		return
	}

	if packetID != 0 {
		fmt.Println("Error handshaking: Unexpected packet type")
		return
	}

	handshake := packets.HandshakePacket{}
	handshake.Read(bufreader)

	fmt.Println("Received handshake", handshake)

	mode := ConnectionMode(handshake.RequestType) // a bit risky?? checking if it's valid first is a good idea

	var packetRegistry *map[int32]func() packets.ServerboundPacket

	for {
		switch mode {
		case Disconnect:
			return
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

		packetID, bufreader, payload, err := packets.ReadPacket(conn)
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

		if err := packet.Read(bufreader); err != nil {
			fmt.Println("Failed to read packet content:", err)
			return
		}

		handler(&mode, packet, conn)
	}
}

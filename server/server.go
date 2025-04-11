package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mcmockserver/packets"
	"net"
	"sync"
	"time"
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
	Ctx    context.Context
	Cancel context.CancelFunc
	Conn   net.Conn
	Queue  chan *bytes.Buffer
	Mode   ConnectionMode
	Wg     sync.WaitGroup
	once   sync.Once
}

func (s *Session) processQueue(ctx context.Context) {
	defer s.Wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case buf := <-s.Queue:
			_, err := buf.WriteTo(s.Conn)
			if err != nil {
				fmt.Println("processQueue:", err)
			}
		}
	}
}

func (s *Session) Close() {
	s.once.Do(func() {
		s.Cancel()
		// wait for the goroutines to finish
		s.Wg.Wait()

	flush: // flush queue
		for {
			select {
			case buf := <-s.Queue:
				_, err := buf.WriteTo(s.Conn)
				if err != nil {
					fmt.Println("processQueue:", err)
				}
			default:
				break flush
			}
		}

		fmt.Println("Closing Session:", s.Conn.RemoteAddr())
		s.Conn.Close()
		close(s.Queue)
	})
}

func NewSession(conn net.Conn, parentCtx context.Context) *Session {
	ctx, cancel := context.WithCancel(parentCtx)
	queue := make(chan *bytes.Buffer, 16)

	session := Session{
		Ctx:    ctx,
		Cancel: cancel,
		Conn:   conn,
		Queue:  queue,
	}

	session.Wg.Add(1)
	go session.processQueue(ctx)

	return &session
}

func ReadPacket(ctx context.Context, session *Session) (id int32, payload *bytes.Reader, err error) {
	readCh := make(chan struct {
		int32
		*bytes.Reader
	}, 1)

	session.Wg.Add(1)
	go func() {
		defer session.Wg.Done()

		id, payload, err := packets.ReadPacket(session.Conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				go session.Close()
			}
			fmt.Println(err)
			return
		}
		readCh <- struct {
			int32
			*bytes.Reader
		}{id, payload}
	}()

	select {
	case <-ctx.Done():
		session.Conn.SetReadDeadline(time.Now())
		err = ctx.Err()
	case read := <-readCh:
		id = read.int32
		payload = read.Reader
	}

	close(readCh)
	return id, payload, err
}

func HandleSession(session *Session, handler PacketHandler) {
	defer session.Close()

	fmt.Println("Client connected:", session.Conn.RemoteAddr())

	// Handshake
	packetID, bufreader, err := ReadPacket(session.Ctx, session)
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

		packetID, payload, err := ReadPacket(session.Ctx, session)
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

		if err := packet.Read(payload); err != nil {
			fmt.Println("Failed to read packet content:", err)
			return
		}

		handler(session, packet)
	}
}

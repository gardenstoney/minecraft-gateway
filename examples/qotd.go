//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
	"github.com/gardenstoney/minecraft-gateway/protocol/server"
)

var quotes = map[string][]string{
	"ko_kr": {"멈추지 말고 계속 해나가기만 한다면 늦어도 상관없다.", "어디든 가치가 있는 곳으로 가려면 지름길은 없다."},
	"en_us": {"Be yourself; everyone else is already taken.", "So many books, so little time."},
}

func HandlePacket(session *server.Session, packet packets.ServerboundPacket) {
	switch p := packet.(type) {
	case *packets.StatusReqPacket:
		server.DefaultStatusReqPacketHandler(session)

	case *packets.PingReqPacket:
		server.DefaultPingReqPacketHandler(session, p)

	case *packets.LoginStartPacket:
		server.DefaultLoginStartPacketHandler(session, p)

	case *packets.LoginAckPacket:
		server.DefaultLoginAckPacketHandler(session)

	case *packets.ClientInfoPacket:
		q, exists := quotes[string(p.Locale)]
		message := "Your language is not supported :("

		if exists {
			message = q[rand.Intn(len(q))]
		}

		buf := bytes.NewBuffer(make([]byte, 0))
		err := packets.ConfigDisconnectPacket{
			Reason: packets.String(message),
		}.Write(buf)

		if err != nil {
			slog.Error("Failed to write ConfigDisconnect", "error", err, "session", session.Transport.String())
			session.Shutdown()
			return
		}

		go func() {
			select {
			case <-time.After(3 * time.Second):
				session.Transport.Write(buf.Bytes())
				session.Shutdown()
			case <-session.Ctx.Done():
			}
		}()
	}
}

func main() {
	// slog.SetLogLoggerLevel(slog.LevelDebug)
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
	defer listener.Close()

	slog.Info("Server started on port 25565")

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Shutdown on Ctrl+C
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		listener.Close()
	}()

accept:
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				break accept
			default:
				slog.Error("Failed to accept client", "error", err)
				continue accept
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			session := server.NewSession(
				server.NewQueueingTransport(
					server.NewKeepAliveTransport(
						&server.NetTransport{Conn: conn},
						5*time.Second,
						func(t time.Time) []byte {
							buf := bytes.NewBuffer(make([]byte, 0))

							packets.ConfigKeepAlivePacket{
								KeepAliveID: packets.Long(t.Unix()),
							}.Write(buf)

							return buf.Bytes()
						},
					),
				),
				ctx,
			)
			server.HandleSession(session, HandlePacket) // Handle each client in a separate goroutine
		}()
	}

	wg.Wait()
}

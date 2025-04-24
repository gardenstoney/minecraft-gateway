//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"context"
	"fmt"
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
			fmt.Println(err)
			session.Shutdown()
			return
		}

		go func() {
			select {
			case <-time.After(3 * time.Second):
				session.Queue <- buf.Bytes()
				session.Shutdown()
			case <-session.Ctx.Done():
			}
		}()
	}
}

func main() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on port 25565")

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
				fmt.Println("Connection error:", err)
				continue accept
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			session := server.NewSession(conn, ctx)
			server.HandleSession(session, HandlePacket) // Handle each client in a separate goroutine
		}()
	}

	wg.Wait()
}

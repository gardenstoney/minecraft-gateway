//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"mcmockserver/packets"
	"mcmockserver/server"
	"net"
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
			session.Close()
			return
		}

		session.Queue <- buf

		session.Close()
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

	ctx := context.Background()

	for {
		conn, err := listener.Accept() // TODO: set timeout
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}

		session := server.NewSession(conn, ctx)

		go server.HandleSession(session, HandlePacket) // Handle each client in a separate goroutine
	}
}

package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
	"github.com/gardenstoney/minecraft-gateway/protocol/server"
)

func retrieveMainServerStatus(host string, port uint16) (resp packets.StatusRespPacket, err error) {
	var addressBuilder strings.Builder
	addressBuilder.WriteString(host)
	addressBuilder.WriteRune(':')
	addressBuilder.WriteString(strconv.Itoa(int(port)))

	conn, err := net.DialTimeout("tcp", addressBuilder.String(), 2*time.Second)

	if err != nil {
		return resp, err
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	err = packets.HandshakePacket{
		ProtocolVersion: 767,
		ServerAddr:      packets.String(host),
		ServerPort:      packets.UnsignedShort(port),
		RequestType:     packets.VarInt(server.Status),
	}.Write(buf)
	if err != nil {
		return resp, err
	}

	err = packets.StatusReqPacket{}.Write(buf)
	if err != nil {
		return resp, err
	}

	_, err = buf.WriteTo(conn)
	if err != nil {
		return resp, err
	}

	packetID, bufreader, err := packets.ReadPacket(conn)
	if err != nil {
		return resp, err
	}

	if packetID != 0 {
		return resp, errors.New("unexpected packetID")
	}

	err = resp.Read(bufreader)

	return resp, err
}

func main() {
	resp, err := retrieveMainServerStatus("mc.brocoli.dev", 25565)
	fmt.Println(resp)
	fmt.Println(err)
}

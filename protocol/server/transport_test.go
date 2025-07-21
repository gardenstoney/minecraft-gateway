package server

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
)

func TestNetTransportRead_CancelledOnPacketLength(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := NetTransport{Conn: server}

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
	defer cancel()

	_, err := transport.Read(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %q, want %q", err, context.DeadlineExceeded)
	}
}

func TestNetTransportRead_CancelledOnPacketContent(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := NetTransport{Conn: server}

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
	defer cancel()

	go func() {
		buf := bytes.NewBuffer(make([]byte, 0))
		packets.VarInt(5).Write(buf)
		buf.WriteTo(client)
	}()

	_, err := transport.Read(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %q, want %q", err, context.DeadlineExceeded)
	}
}

func TestNetTransportRead_Success(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := NetTransport{Conn: server}

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
	defer cancel()

	go func() {
		buf := bytes.NewBuffer(make([]byte, 0))
		packets.VarInt(4).Write(buf)
		packets.Int(5).Write(buf)
		buf.WriteTo(client)
	}()

	payload, err := transport.Read(ctx)

	if err != nil {
		t.Fatalf("unexpected error on read: %q", err)
	}

	var got packets.Int
	got.Read(bytes.NewReader(payload))
	var want int32 = 5
	if int32(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
)

// Low level communication interface dealing with send queues, compression, and encryption
//
// Transporters can be coupled with each other to add functionality to the layer, such as
// QueueingTransport -> EncryptTransport -> CompressTransport -> NetTransport
type Transporter interface {
	Read(context.Context) ([]byte, error)
	Write(buf []byte) error
	Close() error
	fmt.Stringer
}

// Transporter wrapper for net.Conn
type NetTransport struct {
	Conn net.Conn
}

// Read a packet from net.Conn, cancelable via context
//
// ctx.Err() is prioritized to be returned if it's not nil.
func (t *NetTransport) Read(ctx context.Context) (payload []byte, err error) {
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			_ = t.Conn.SetReadDeadline(time.Now()) // force read to unblock
		case <-done:
		}
	}()

	payloadLength, err := packets.ReadVarint(t.Conn)
	if err != nil {
		if e := ctx.Err(); e != nil {
			err = e
		}
		return payload, err
	}

	payload = make([]byte, payloadLength)
	_, err = io.ReadFull(t.Conn, payload)

	if e := ctx.Err(); e != nil {
		err = e
	}
	return payload, err
}

func (t *NetTransport) Write(buf []byte) error {
	_, err := t.Conn.Write(buf)
	return err
}

func (t *NetTransport) Close() error {
	return t.Conn.Close()
}

func (t NetTransport) String() string {
	return t.Conn.RemoteAddr().String()
}

// Intermediary transporter for setting up send queue
// to make writes in multiple goroutines safe
type QueueingTransport struct {
	inner Transporter
	queue chan []byte
	done  chan struct{}
}

func NewQueueingTransport(inner Transporter) *QueueingTransport {
	queue := make(chan []byte, 16)
	done := make(chan struct{})

	go func(queue chan []byte, done chan struct{}) {
		for {
			select {
			case <-done:
				return
			case buf := <-queue:
				err := inner.Write(buf)
				if err != nil {
					slog.Error("Send queue error", "error", err, "transport", "QueueingTransport", "source", inner.String())
				}
			}
		}
	}(queue, done)

	return &QueueingTransport{inner, queue, done}
}

func (t *QueueingTransport) Read(ctx context.Context) ([]byte, error) {
	return t.inner.Read(ctx)
}

// Enqueues the given buf
//
// Always returns nil for error.
// May block if the queue channel is full and isn't dequeued for some reason.
func (t *QueueingTransport) Write(buf []byte) error {
	t.queue <- buf
	return nil
}

// End queueing loop goroutine, flush the queue and close inner transporter
func (t *QueueingTransport) Close() error {
	t.done <- struct{}{} // blocks until it's received.
	t.flush()

	close(t.queue)
	close(t.done)

	return t.inner.Close()
}

func (t *QueueingTransport) flush() (err error) {
	for {
		select {
		case buf := <-t.queue:
			err = t.inner.Write(buf)
			if err != nil {
				slog.Error("Flush queue error", "error", err, "transport", "QueueingTransport", "source", t.String())
			}
		default:
			return
		}
	}
}

func (t QueueingTransport) String() string {
	return t.inner.String()
}

// Convenience transporter for handling keep alives in transport layer
//
// Sends keep alive when write isn't called for a given time.
type KeepAliveTransport struct {
	inner      Transporter
	resetTimer chan struct{}
	done       chan struct{}
}

func NewKeepAliveTransport(
	inner Transporter,
	writeIdleTimeout time.Duration,
	keepAliveFunc func(time.Time) []byte,
) *KeepAliveTransport {

	resetTimer := make(chan struct{})
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-resetTimer:
			case t := <-time.After(writeIdleTimeout):
				err := inner.Write(keepAliveFunc(t))
				if err != nil {
					slog.Error("KeepAlive write error", "error", err, "transport", "QueueingTransport", "source", t.String())
				}
			}
		}
	}()

	return &KeepAliveTransport{inner, resetTimer, done}
}

func (t *KeepAliveTransport) Read(ctx context.Context) ([]byte, error) {
	return t.inner.Read(ctx)
}

// Write to inner transporter and reset timer if the write succeeds
func (t *KeepAliveTransport) Write(buf []byte) error {
	err := t.inner.Write(buf)
	if err == nil {
		t.resetTimer <- struct{}{}
	}

	return err
}

// End timer goroutine and close inner transporter
func (t *KeepAliveTransport) Close() error {
	t.done <- struct{}{} // blocks until it's received.

	close(t.resetTimer)
	close(t.done)

	return t.inner.Close()
}

func (t KeepAliveTransport) String() string {
	return t.inner.String()
}

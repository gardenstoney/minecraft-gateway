package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gardenstoney/minecraft-gateway/protocol/packets"
	"github.com/gardenstoney/minecraft-gateway/protocol/server"
	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var cfg *Config

var ec2Client *ec2.Client

var backgroundCtx context.Context
var topWg sync.WaitGroup

var swtRunning atomic.Bool

var waitingListMu sync.Mutex
var waitingList []*server.Session

func HandlePacket(session *server.Session, packet packets.ServerboundPacket) {
	switch p := packet.(type) {
	case *packets.StatusReqPacket:
		// TODO: relay the statusreqpacket from main server when it's online
		server.DefaultStatusReqPacketHandler(session)

	case *packets.PingReqPacket:
		server.DefaultPingReqPacketHandler(session, p)

	case *packets.LoginStartPacket:
		if !slices.Contains(cfg.Whitelist, uuid.UUID(p.PlayerUUID)) {
			session.Shutdown()
			return
		}
		server.DefaultLoginStartPacketHandler(session, p)

	case *packets.LoginAckPacket:
		server.DefaultLoginAckPacketHandler(session)

	case *packets.ClientInfoPacket:
		instance, err := retrieveInstance(cfg.InstanceID)
		if err != nil {
			buf := bytes.NewBuffer(make([]byte, 0))
			packets.ConfigDisconnectPacket{Reason: "An error occured."}.Write(buf)

			session.Transport.Write(buf.Bytes())
			session.Shutdown()
			fmt.Println(err)
			return
		}

		// transfer if server is ready
		if *instance.State.Code == 16 {
			_, err = retrieveMainServerStatus(*instance.PublicDnsName, cfg.Port)
			if err == nil {
				buf := bytes.NewBuffer(make([]byte, 0))
				packets.ConfigTransferPacket{
					Host: packets.String(*instance.PublicIpAddress),
					Port: packets.VarInt(cfg.Port),
				}.Write(buf)

				session.Transport.Write(buf.Bytes())
				session.Shutdown()
				return
			}
		}

		// run StartWaitTransfer goroutine just once if the instance is offline
		if *instance.State.Code == 80 || *instance.State.Code == 64 {
			if swtRunning.CompareAndSwap(false, true) {
				topWg.Add(1)
				go func(state int32) {
					defer topWg.Done()

					// wait for the instance state to become 'stopped' when the instance is 'stopping'
					if state == 64 {
						instance, err = waitForStateChange(backgroundCtx, 80)
						if err != nil { // Context canceled
							return
						}
					}
					StartWaitTransfer()
					swtRunning.Store(false)
				}(*instance.State.Code)
			}
		}

		// register the session to the waitingList
		waitingListMu.Lock()
		waitingList = append(waitingList, session)
		waitingListMu.Unlock()

		session.Wg.Add(1)
		go func(s *server.Session) {
			defer s.Wg.Done()

			<-s.Ctx.Done()

			waitingListMu.Lock()
			for i := 0; i < len(waitingList); i++ {
				if waitingList[i] == s {
					waitingList[i] = waitingList[len(waitingList)-1]
					waitingList = waitingList[:len(waitingList)-1]
					break
				}
			}
			waitingListMu.Unlock()
		}(session)
	}
}

func StartWaitTransfer() {
	// Start
	_, err := ec2Client.StartInstances(
		backgroundCtx,
		&ec2.StartInstancesInput{InstanceIds: []string{cfg.InstanceID}},
	)
	if err != nil { // ctx.err when backgroundCtx cancels during startinstances
		buf := bytes.NewBuffer(make([]byte, 0))
		packets.ConfigDisconnectPacket{
			Reason: "An error occured.",
		}.Write(buf)

		broadcastPacketAndClean(buf.Bytes())
		fmt.Println(err)
		return
	}

	// Wait
	if err := waitForMainServer(backgroundCtx); err != nil {
		return // Context canceled
	}

	// Transfer
	instance, err := retrieveInstance(cfg.InstanceID)
	if err != nil {
		buf := bytes.NewBuffer(make([]byte, 0))
		packets.ConfigDisconnectPacket{
			Reason: "An error occured.",
		}.Write(buf)

		broadcastPacketAndClean(buf.Bytes())
		fmt.Println(err)
		return
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	packets.ConfigTransferPacket{
		Host: packets.String(*instance.PublicIpAddress),
		Port: packets.VarInt(cfg.Port),
	}.Write(buf)

	broadcastPacketAndClean(buf.Bytes())
}

func waitForStateChange(ctx context.Context, expect int32) (instance *types.Instance, err error) {
	for {
		c := make(chan int32)

		go func(c chan int32) {
			defer close(c)

			instance, err = retrieveInstance(cfg.InstanceID)
			if err != nil {
				fmt.Println(err)
				c <- -1
			} else {
				c <- *instance.State.Code
			}
		}(c)

		select {
		case code := <-c:
			if code == expect {
				return instance, nil
			}
		case <-ctx.Done():
			<-c
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func waitForMainServer(ctx context.Context) error {
	instance, err := waitForStateChange(ctx, 16)
	if err != nil {
		return err
	}

	for {
		c := make(chan error)

		go func(c chan error) {
			defer close(c)

			_, err := retrieveMainServerStatus(*instance.PublicDnsName, cfg.Port)
			c <- err
		}(c)

		select {
		case err := <-c:
			if err == nil {
				return nil
			}
		case <-ctx.Done():
			<-c
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func broadcastPacketAndClean(buf []byte) {
	waitingListMu.Lock()
	for _, s := range waitingList {
		s.Transport.Write(buf)

		topWg.Add(1)
		go func() { // prevent deadlock with waitinglist clean goroutine when session ends
			defer topWg.Done()
			s.Shutdown()
		}()
	}
	waitingList = waitingList[:0]
	waitingListMu.Unlock()
}

func retrieveInstance(id string) (*types.Instance, error) {
	output, err := ec2Client.DescribeInstances(
		context.TODO(),
		&ec2.DescribeInstancesInput{InstanceIds: []string{id}},
	)
	if err != nil {
		return nil, err
	}

	return &output.Reservations[0].Instances[0], nil
}

func retrieveMainServerStatus(host string, port uint16) (resp packets.StatusRespPacket, err error) {
	var addressBuilder strings.Builder
	addressBuilder.WriteString(host)
	addressBuilder.WriteRune(':')
	addressBuilder.WriteString(strconv.Itoa(int(port)))

	conn, err := net.DialTimeout("tcp", addressBuilder.String(), 1*time.Second)

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
	var err error
	cfg, err = LoadConfig("config.yaml")
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	var ec2cfg aws.Config
	ec2cfg, err = config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ec2Client = ec2.NewFromConfig(ec2cfg)

	waitingList = make([]*server.Session, 0)

	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on port 25565")

	var cancel context.CancelFunc
	backgroundCtx, cancel = context.WithCancel(context.Background())

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
			case <-backgroundCtx.Done():
				break accept
			default:
				fmt.Println("Connection error:", err)
				continue accept
			}
		}

		topWg.Add(1)
		go func() {
			defer topWg.Done()
			fmt.Println("Client connected:", conn.RemoteAddr())
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
				backgroundCtx,
			)

			server.HandleSession(session, HandlePacket) // Handle each session in a separate goroutine
			// fmt.Println("done handling session")
		}()
	}

	topWg.Wait()
}

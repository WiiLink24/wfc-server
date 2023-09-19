package master

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var (
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions = map[uint32]*Session{}
	mutex    = sync.RWMutex{}
)

const (
	CommandQuery            = 0x00
	CommandChallenge        = 0x01
	CommandEcho             = 0x02
	CommandHeartbeat        = 0x03
	CommandAddError         = 0x04
	CommandEchoResponse     = 0x05
	CommandClientMessage    = 0x06
	CommandClientMessageAck = 0x07
	CommandKeepAlive        = 0x08
	CommandAvailable        = 0x09
	CommandClientRegistered = 0x0A
)

func StartServer() {
	conn, err := net.ListenPacket("udp", ":27900")
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		go handleConnection(conn, addr, buf)
	}
}

func handleConnection(conn net.PacketConn, addr net.Addr, buffer []byte) {
	if buffer[0] != 9 {
		addSession(addr, buffer)
	}

	switch buffer[0] {
	case CommandQuery:
		logging.Notice("MASTER", "Command:", aurora.Yellow("QUERY").String())
		break

	case CommandChallenge:
		logging.Notice("MASTER", "Command:", aurora.Yellow("CHALLENGE").String())
		sessionId := binary.BigEndian.Uint32(buffer[1:5])
		conn.WriteTo(createResponseHeader(CommandClientRegistered, sessionId), addr)
		break

	case CommandEcho:
		logging.Notice("MASTER", "Command:", aurora.Yellow("ECHO").String())
		break

	case CommandHeartbeat:
		logging.Notice("MASTER", "Command:", aurora.Yellow("HEARTBEAT").String())
		heartbeat(conn, addr, buffer)
		break

	case CommandAddError:
		logging.Notice("MASTER", "Command:", aurora.Yellow("ADDERROR").String())
		break

	case CommandEchoResponse:
		logging.Notice("MASTER", "Command:", aurora.Yellow("ECHO_RESPONSE").String())
		break

	case CommandClientMessage:
		logging.Notice("MASTER", "Command:", aurora.Yellow("CLIENT_MESSAGE").String())
		return

	case CommandClientMessageAck:
		logging.Notice("MASTER", "Command:", aurora.Yellow("CLIENT_MESSAGE_ACK").String())
		return

	case CommandKeepAlive:
		logging.Notice("MASTER", "Command:", aurora.Yellow("KEEPALIVE").String())
		return

	case CommandAvailable:
		logging.Notice("MASTER", "Command:", aurora.Yellow("AVAILABLE").String())
		conn.WriteTo(createResponseHeader(CommandAvailable, 0), addr)
		break

	case CommandClientRegistered:
		logging.Notice("MASTER", "Command:", aurora.Yellow("QUERY").String())
		break

	default:
		logging.Notice("MASTER", "Unknown command:", aurora.Yellow(buffer[0]).String())
		return
	}
}

func createResponseHeader(command byte, sessionId uint32) []byte {
	return binary.BigEndian.AppendUint32([]byte{0xfe, 0xfd, command}, sessionId)
}

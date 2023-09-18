package master

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var (
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions = map[uint32]*Session{}
	mutex    = sync.RWMutex{}
)

const (
	Command_QUERY              = 0x00
	Command_CHALLENGE          = 0x01
	Command_ECHO               = 0x02
	Command_HEARTBEAT          = 0x03
	Command_ADDERROR           = 0x04
	Command_ECHO_RESPONSE      = 0x05
	Command_CLIENT_MESSAGE     = 0x06
	Command_CLIENT_MESSAGE_ACK = 0x07
	Command_KEEPALIVE          = 0x08
	Command_AVAILABLE          = 0x09
	Command_CLIENT_REGISTERED  = 0x0A
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
	case Command_QUERY:
		logging.Notice("MASTER", "Command:", aurora.Yellow("QUERY").String())
		break

	case Command_CHALLENGE:
		logging.Notice("MASTER", "Command:", aurora.Yellow("CHALLENGE").String())
		// sessionId := binary.BigEndian.Uint32(buffer[1:5])
		// conn.WriteTo(createResponseHeader(Command_CLIENT_REGISTERED, sessionId), addr)
		break

	case Command_ECHO:
		logging.Notice("MASTER", "Command:", aurora.Yellow("ECHO").String())
		break

	case Command_HEARTBEAT:
		logging.Notice("MASTER", "Command:", aurora.Yellow("HEARTBEAT").String())
		heartbeat(conn, addr, buffer)
		break

	case Command_ADDERROR:
		logging.Notice("MASTER", "Command:", aurora.Yellow("ADDERROR").String())
		break

	case Command_ECHO_RESPONSE:
		logging.Notice("MASTER", "Command:", aurora.Yellow("ECHO_RESPONSE").String())
		break

	case Command_CLIENT_MESSAGE:
		logging.Notice("MASTER", "Command:", aurora.Yellow("CLIENT_MESSAGE").String())
		return

	case Command_CLIENT_MESSAGE_ACK:
		logging.Notice("MASTER", "Command:", aurora.Yellow("CLIENT_MESSAGE_ACK").String())
		return

	case Command_KEEPALIVE:
		logging.Notice("MASTER", "Command:", aurora.Yellow("KEEPALIVE").String())
		return

	case Command_AVAILABLE:
		logging.Notice("MASTER", "Command:", aurora.Yellow("AVAILABLE").String())
		conn.WriteTo(createResponseHeader(Command_AVAILABLE, 0), addr)
		break

	case Command_CLIENT_REGISTERED:
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

func sendChallenge(conn net.PacketConn, addr net.Addr, sessionId uint32) {
	addrString := strings.Split(addr.String(), ":")

	// Generate challenge and send to server
	var hexIP string
	for _, i := range strings.Split(addrString[0], ".") {
		val, err := strconv.ParseUint(i, 10, 64)
		if err != nil {
			panic(err)
		}

		hexIP += fmt.Sprintf("%02X", val)
	}

	port, err := strconv.ParseUint(addrString[1], 10, 64)
	if err != nil {
		panic(err)
	}

	hexPort := fmt.Sprintf("%04X", port)

	challenge := common.RandomString(6) + "00" + hexIP + hexPort
	mutex.Lock()
	session := sessions[sessionId]
	session.Challenge = challenge
	mutex.Unlock()

	response := createResponseHeader(Command_CHALLENGE, sessionId)
	response = append(response, []byte(challenge)...)
	response = append(response, 0)

	conn.WriteTo(response, addr)
}

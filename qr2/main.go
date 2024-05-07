package qr2

import (
	"encoding/binary"
	"net"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

const (
	QueryRequest            = 0x00
	ChallengeRequest        = 0x01
	EchoRequest             = 0x02
	HeartbeatRequest        = 0x03
	AddErrorRequest         = 0x04
	EchoResponseRequest     = 0x05
	ClientMessageRequest    = 0x06
	ClientMessageAckRequest = 0x07
	KeepAliveRequest        = 0x08
	AvailableRequest        = 0x09
	ClientRegisteredReply   = 0x0A

	ClientExploitReply = 0x10
)

var masterConn net.PacketConn

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	address := *config.GameSpyAddress + ":27900"
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		panic(err)
	}

	masterConn = conn

	if reload {
		loadSessions()
		loadLogins()
		loadGroups()
		logging.Info("QR2", "Loaded", aurora.Cyan(len(sessions)), "sessions")
	}

	go func() {
		// Close the listener when the application closes.
		defer conn.Close()
		logging.Notice("QR2", "Listening on", aurora.BrightCyan(address))

		for {
			buf := make([]byte, 1024)
			_, addr, err := conn.ReadFrom(buf)
			if err != nil {
				continue
			}

			go handleConnection(conn, *addr.(*net.UDPAddr), buf)
		}
	}()
}

func Shutdown() {
	// Lock indefinitely
	mutex.Lock()

	err := saveSessions()
	if err != nil {
		logging.Error("QR2", "Failed to save sessions:", err)
	}

	logging.Info("QR2", "Saved", aurora.Cyan(len(sessions)), "sessions")

	err = saveLogins()
	if err != nil {
		logging.Error("QR2", "Failed to save logins:", err)
	}

	logging.Info("QR2", "Saved", aurora.Cyan(len(logins)), "logins")

	err = saveGroups()
	if err != nil {
		logging.Error("QR2", "Failed to save groups:", err)
	}
}

func handleConnection(conn net.PacketConn, addr net.UDPAddr, buffer []byte) {
	packetType := buffer[0]
	moduleName := "QR2:" + addr.String()

	var session *Session
	if packetType != HeartbeatRequest && packetType != AvailableRequest {
		mutex.Lock()

		var ok bool
		session, ok = sessions[makeLookupAddr(addr.String())]
		if !ok {
			mutex.Unlock()
			logging.Error(moduleName, "Cannot find session for this IP address")
			return
		}

		session.SessionID = binary.BigEndian.Uint32(buffer[1:5])

		mutex.Unlock()
	}

	switch packetType {
	case QueryRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("QUERY"))

	case ChallengeRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("CHALLENGE"))

		mutex.Lock()
		if session.Challenge != "" {
			// TODO: Verify the challenge
			session.Authenticated = true
			mutex.Unlock()

			conn.WriteTo(createResponseHeader(ClientRegisteredReply, session.SessionID), &addr)
		} else {
			mutex.Unlock()
		}

	case EchoRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ECHO"))

	case HeartbeatRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("HEARTBEAT"))
		heartbeat(moduleName, conn, addr, buffer)

	case AddErrorRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ADDERROR"))

	case EchoResponseRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ECHO_RESPONSE"))

	case ClientMessageRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_MESSAGE"))
		return

	case ClientMessageAckRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_MESSAGE_ACK"))

		// In case ClientExploitReply is lost, this can be checked as well
		// This would be sent either after the payload is downloaded, or the client is already patched
		session.ExploitReceived = true
		if login := session.login; login != nil {
			login.NeedsExploit = false
		}

		session.messageAckWaker.Assert()
		return

	case KeepAliveRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("KEEPALIVE"))
		conn.WriteTo(createResponseHeader(KeepAliveRequest, 0), &addr)

		session.LastKeepAlive = time.Now().Unix()
		return

	case AvailableRequest:
		logging.Info("QR2", "Command:", aurora.Yellow("AVAILABLE"))
		conn.WriteTo(createResponseHeader(AvailableRequest, 0), &addr)
		return

	case ClientRegisteredReply:
		logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_REGISTERED"))

	case ClientExploitReply:
		logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_EXPLOIT_ACK"))

		session.ExploitReceived = true
		if login := session.login; login != nil {
			login.NeedsExploit = false
		}

	default:
		logging.Error(moduleName, "Unknown command:", aurora.Yellow(buffer[0]))
		return
	}
}

func createResponseHeader(command byte, sessionId uint32) []byte {
	return binary.BigEndian.AppendUint32([]byte{0xfe, 0xfd, command}, sessionId)
}

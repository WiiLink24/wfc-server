package qr2

import (
	"encoding/binary"
	"net"
	"sync"
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

	ClientExploitReply  = 0x10
	ClientKickPeerOrder = 0x11
)

var (
	masterConn net.PacketConn
	inShutdown = false
	waitGroup  = sync.WaitGroup{}
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	address := *config.GameSpyAddress + ":27900"
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		panic(err)
	}

	masterConn = conn
	inShutdown = false

	if reload {
		// Load state - ensure mutex acquired only once for all operations
		mutex.Lock()
		
		err := loadSessions()
		if err != nil {
			mutex.Unlock()
			panic(err)
		}
		logging.Notice("QR2", "Loaded", aurora.Cyan(len(sessions)), "sessions")

		err = loadLogins()
		if err != nil {
			mutex.Unlock()
			panic(err)
		}
		logging.Notice("QR2", "Loaded", aurora.Cyan(len(logins)), "logins")

		err = loadGroups()
		if err != nil {
			mutex.Unlock()
			panic(err)
		}
		logging.Notice("QR2", "Loaded", aurora.Cyan(len(groups)), "groups")
		
		mutex.Unlock()
	}

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		// Close the listener when the application closes.
		defer conn.Close()
		logging.Notice("QR2", "Listening on", aurora.BrightCyan(address))

		for {
			if inShutdown {
				return
			}

			buf := make([]byte, 1024)
			n, addr, err := conn.ReadFrom(buf)
			if err != nil || n == 0 {
				continue
			}

			waitGroup.Add(1)

			go handleConnection(conn, *addr.(*net.UDPAddr), buf[:n])
		}
	}()
}

func Shutdown() {
	inShutdown = true
	masterConn.Close()
	waitGroup.Wait()

	// Lock mutex only once for all operations
	mutex.Lock()
	defer mutex.Unlock()

	err := saveSessions()
	if err != nil {
		logging.Error("QR2", "Failed to save sessions:", err)
	}
	logging.Notice("QR2", "Saved", aurora.Cyan(len(sessions)), "sessions")

	err = saveLogins()
	if err != nil {
		logging.Error("QR2", "Failed to save logins:", err)
	}
	logging.Notice("QR2", "Saved", aurora.Cyan(len(logins)), "logins")

	err = saveGroups()
	if err != nil {
		logging.Error("QR2", "Failed to save groups:", err)
	}
	logging.Notice("QR2", "Saved", aurora.Cyan(len(groups)), "groups")
}

func handleConnection(conn net.PacketConn, addr net.UDPAddr, buffer []byte) {
	defer waitGroup.Done()

	packetType := buffer[0]
	moduleName := "QR2:" + addr.String()
	
	// Handle special packet types that don't need locking first
	if packetType == AvailableRequest {
		logging.Info("QR2", "Command:", aurora.Yellow("AVAILABLE"))
		conn.WriteTo(createResponseHeader(AvailableRequest, 0), &addr)
		return
	}
	
	if packetType == HeartbeatRequest {
		// heartbeat handles its own locking
		heartbeat(moduleName, conn, addr, buffer)
		return
	}

	// For all other packet types, acquire the mutex
	mutex.Lock()
	defer mutex.Unlock()

	var session *Session
	lookupAddr := makeLookupAddr(addr.String())
	var ok bool
	
	// Don't check for session for KeepAlive requests as they can come before a session is established
	if packetType != KeepAliveRequest {
		session, ok = sessions[lookupAddr]
		if !ok {
			logging.Error(moduleName, "Cannot find session for this IP address")
			return
		}

		session.SessionID = binary.BigEndian.Uint32(buffer[1:5])
	}

	switch packetType {
	case QueryRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("QUERY"))

	case ChallengeRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("CHALLENGE"))
		
		if session.Challenge != "" {
			// TODO: Verify the challenge
			session.Authenticated = true
			
			// Store session ID before releasing mutex for writeTo
			sessionID := session.SessionID
			mutex.Unlock()

			conn.WriteTo(createResponseHeader(ClientRegisteredReply, sessionID), &addr)
			
			// Re-acquire mutex afterward since this function has a deferred unlock
			mutex.Lock()
		}

	case EchoRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ECHO"))

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

		// Wake up any waiting threads
		ackWaker := session.messageAckWaker
		mutex.Unlock()
		
		ackWaker.Assert()
		
		// Re-acquire mutex for deferred unlock
		mutex.Lock()
		return

	case KeepAliveRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("KEEPALIVE"))
		
		// Temporarily release mutex for network I/O
		mutex.Unlock()
		conn.WriteTo(createResponseHeader(KeepAliveRequest, 0), &addr)
		mutex.Lock()

		session, ok = sessions[lookupAddr]
		if ok {
			session.LastKeepAlive = time.Now().Unix()
		}
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
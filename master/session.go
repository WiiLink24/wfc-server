package master

import (
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
	"strings"
	"time"
	"wwfc/logging"
)

const (
	ClientNoEndian = iota
	ClientBigEndian
	ClientLittleEndian
)

type Session struct {
	SessionID     uint32
	Addr          net.Addr
	Challenge     string
	Authenticated bool
	LastKeepAlive int64
	Endianness    byte // Some fields depend on the client's endianness
	Data          map[string]string
}

// Remove a session.
func removeSession(sessionId uint32) {
	delete(sessions, sessionId)
}

// Update session data, creating the session if it doesn't exist. Returns a copy of the session data.
func setSessionData(sessionId uint32, payload map[string]string) Session {
	mutex.Lock()
	defer mutex.Unlock()

	session, exists := sessions[sessionId]
	if !exists {
		logging.Notice("MASTER", "Creating session", aurora.Cyan(sessionId).String())
		data := Session{
			SessionID:     sessionId,
			Challenge:     "",
			Authenticated: false,
			LastKeepAlive: time.Now().Unix(),
			Endianness:    ClientNoEndian,
			Data:          payload,
		}
		sessions[sessionId] = &data

		return data
	}

	session.Data = payload
	session.LastKeepAlive = time.Now().Unix()

	return *session
}

// Get a copy of the list of servers
func GetSessionServers() []map[string]string {
	var servers []map[string]string
	currentTime := time.Now().Unix()

	mutex.Lock()
	defer mutex.Unlock()
	for _, session := range sessions {
		if !session.Authenticated {
			continue
		}

		// If the last keep alive was over a minute ago then consider the server unreachable
		if session.LastKeepAlive < currentTime-60 {
			continue
		}

		servers = append(servers, session.Data)
	}

	return servers
}

func SendClientMessage(destIP string, message []byte) {
	var rawIP int
	for i, s := range strings.Split(strings.Split(destIP, ":")[0], ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		rawIP |= val << (24 - i*8)
	}

	// TODO: Check if this handles negative numbers correctly
	destIPIntStr := strconv.FormatInt(int64(int32(rawIP)), 10)

	currentTime := time.Now().Unix()

	mutex.Lock()
	// Find the session with the IP
	for _, session := range sessions {
		if !session.Authenticated {
			continue
		}

		// If the last keep alive was over a minute ago then consider the server unreachable
		if session.LastKeepAlive < currentTime-60 {
			continue
		}

		if session.Data["publicip"] == destIPIntStr {
			// Found the client, now send the message
			payload := createResponseHeader(ClientMessageRequest, session.SessionID)
			mutex.Unlock()

			payload = append(payload, []byte{0, 0, 0, 0}...)
			binary.BigEndian.PutUint32(payload[len(payload)-4:], uint32(time.Now().Unix()))
			payload = append(payload, message...)

			destIPAddr, err := net.ResolveUDPAddr("udp", destIP)
			if err != nil {
				panic(err)
			}

			logging.Notice("MASTER", "Sending message...")
			masterConn.WriteTo(payload, destIPAddr)
			return
		}
	}
	mutex.Unlock()

	logging.Notice("MASTER", "Could not find destination server")
}

package master

import (
	"github.com/logrusorgru/aurora/v3"
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
	mutex.Lock()
	defer mutex.Unlock()

	currentTime := time.Now().Unix()

	var servers []map[string]string
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

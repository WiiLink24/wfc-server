package qr2

import (
	"github.com/logrusorgru/aurora/v3"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"
)

const (
	ClientNoEndian = iota
	ClientBigEndian
	ClientLittleEndian
)

type Session struct {
	SessionID     uint32
	SearchID      uint64
	Addr          net.Addr
	Challenge     string
	Authenticated bool
	LastKeepAlive int64
	Endianness    byte // Some fields depend on the client's endianness
	Data          map[string]string
	PacketCount   uint32
}

var (
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions          = map[uint32]*Session{}
	sessionByPublicIP = map[uint64]*Session{}
	sessionBySearchID = map[uint64]*Session{}
	mutex             = sync.RWMutex{}
)

// Remove a session.
func removeSession(sessionId uint32) {
	mutex.Lock()
	defer mutex.Unlock()

	if sessions[sessionId] == nil {
		return
	}

	// Delete search ID lookup
	delete(sessionBySearchID, sessions[sessionId].SearchID)

	// Delete public IP lookup
	ip, port := common.IPFormatToInt(sessions[sessionId].Addr.String())
	lookupIP := (uint64(port) << 32) | uint64(uint32(ip))
	delete(sessionByPublicIP, lookupIP)

	delete(sessions, sessionId)
}

// Update session data, creating the session if it doesn't exist. Returns a copy of the session data.
func setSessionData(sessionId uint32, payload map[string]string, addr net.Addr) (Session, bool) {
	moduleName := "QR2:" + strconv.FormatInt(int64(sessionId), 10)

	// Perform sanity checks on the session data. This is a mess but

	// This is an internal error that should not happen
	publicIP := ""
	var ok bool
	if publicIP, ok = payload["publicip"]; !ok || publicIP == "0" {
		logging.Error(moduleName, "Missing publicip in session data")
		return Session{}, false
	}

	newPID, newPIDValid := payload["dwc_pid"]

	// Moving into performing operations on the session data, so lock the mutex
	mutex.Lock()
	defer mutex.Unlock()
	session, sessionExists := sessions[sessionId]

	if sessionExists && session.Data["publicip"] != publicIP {
		logging.Error(moduleName, "Public IP mismatch")
		return Session{}, false
	}

	if newPIDValid {
		var oldPID string
		oldPIDValid := false

		if sessionExists {
			if oldPID, oldPIDValid = session.Data["dwc_pid"]; oldPIDValid && newPID != oldPID {
				logging.Error(moduleName, "New dwc_pid mismatch: new:", aurora.Cyan(newPID), "old:", aurora.Cyan(oldPID))
				return Session{}, false
			}
		}

		if !oldPIDValid {
			// Setting a new PID so validate it
			profileID, err := strconv.ParseUint(newPID, 10, 32)
			if err != nil {
				logging.Error(moduleName, "Invalid dwc_pid value:", aurora.Cyan(newPID))
				return Session{}, false
			}

			// Reformat dwc_pid string
			newPID = strconv.FormatUint(uint64(profileID), 10)
			payload["dwc_pid"] = newPID

			// Lookup the profile ID in GPCM and verify it's logged in.
			// Maybe we don't need this? It relies on GPCM being hosted in the same application, and
			// makes GPCM a dependency of QR2. Perhaps we could use the database.
			gpcmIP := gpcm.GetSessionIP(uint32(profileID))
			if gpcmIP == "" {
				logging.Error(moduleName, "Provided dwc_pid is not logged in:", aurora.Cyan(newPID))
				return Session{}, false
			}

			gpcmIPStr, _ := common.IPFormatToString(gpcmIP)
			if gpcmIPStr != publicIP {
				logging.Error(moduleName, "Caller public IP does not match GPCM session")
				return Session{}, false
			}

			// Constraint: only one session can exist with a profile ID
			outdated := []uint32{}
			for sessionID, otherSession := range sessions {
				if otherPID, ok := otherSession.Data["dwc_pid"]; !ok || otherPID != newPID {
					continue
				}

				// Remove old sessions with the PID
				outdated = append(outdated, sessionID)
			}

			for _, sessionID := range outdated {
				logging.Notice(moduleName, "Removing outdated session", aurora.BrightCyan(sessionID), "with PID", aurora.Cyan(newPID))
				delete(sessions, sessionID)
			}

			logging.Notice(moduleName, "Opened session with PID", aurora.Cyan(newPID))
		}
	}

	if !sessionExists {
		logging.Notice(moduleName, "Creating session", aurora.Cyan(sessionId).String())
		data := Session{
			SessionID:     sessionId,
			Addr:          addr,
			Challenge:     "",
			Authenticated: false,
			LastKeepAlive: time.Now().Unix(),
			Endianness:    ClientNoEndian,
			Data:          payload,
			PacketCount:   0,
		}

		// Set search ID
		for {
			searchID := uint64(rand.Int63n(0x400 << 32))
			if _, exists := sessionBySearchID[searchID]; !exists {
				data.SearchID = searchID
				data.Data["+searchid"] = strconv.FormatUint(searchID, 10)
				sessionBySearchID[searchID] = &data
				break
			}
		}

		// Set public IP lookup
		ip, port := common.IPFormatToInt(addr.String())
		lookupIP := (uint64(port) << 32) | uint64(uint32(ip))
		sessionByPublicIP[lookupIP] = &data

		sessions[sessionId] = &data
		return data, true
	}

	payload["+searchid"] = session.Data["+searchid"]
	session.Data = payload
	session.LastKeepAlive = time.Now().Unix()
	return *session, true
}

// Get a copy of the list of servers
func GetSessionServers() []map[string]string {
	servers := []map[string]string{}
	unreachable := []uint32{}
	currentTime := time.Now().Unix()

	mutex.Lock()
	defer mutex.Unlock()
	for _, session := range sessions {
		// If the last keep alive was over a minute ago then consider the server unreachable
		if session.LastKeepAlive < currentTime-60 {
			// If the last keep alive was over an hour ago then remove the server
			if session.LastKeepAlive < currentTime-((60*60)*1) {
				unreachable = append(unreachable, session.SessionID)
			}
			continue
		}

		if !session.Authenticated {
			continue
		}

		servers = append(servers, session.Data)
	}

	// Remove unreachable sessions
	for _, sessionID := range unreachable {
		logging.Notice("QR2", "Removing unreachable session", aurora.BrightCyan(sessionID))
		delete(sessions, sessionID)
	}

	return servers
}

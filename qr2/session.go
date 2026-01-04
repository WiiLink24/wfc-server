package qr2

import (
	"encoding/gob"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/linkdata/deadlock"
	"github.com/logrusorgru/aurora/v3"
	"gvisor.dev/gvisor/pkg/sleep"
)

const (
	ClientLittleEndian = 0
	ClientBigEndian    = 1
	ClientNoEndian     = 2
)

type Session struct {
	SessionID       uint32
	SearchID        uint64
	Addr            net.UDPAddr
	Challenge       string
	Authenticated   bool
	login           *LoginInfo
	ExploitReceived bool
	LastKeepAlive   int64
	Endianness      byte // Some fields depend on the client's endianness
	Data            map[string]string
	PacketCount     uint32
	Reservation     common.MatchCommandData
	ReservationID   uint64
	messageMutex    *deadlock.Mutex
	messageAckWaker *sleep.Waker
	groupPointer    *Group
	GroupName       string
}

var (
	sessions          = map[uint64]*Session{}
	sessionBySearchID = map[uint64]*Session{}
	mutex             = deadlock.Mutex{}
)

// Remove a session. Expects the global mutex to already be locked.
func removeSession(addr uint64) {
	session := sessions[addr]
	if session == nil {
		return
	}

	session.messageAckWaker.Assert()

	if session.groupPointer != nil {
		session.removeFromGroup()
	}

	if session.login != nil {
		session.login.session = nil
		session.login = nil
	}

	// Delete search ID lookup
	delete(sessionBySearchID, sessions[addr].SearchID)

	delete(sessions, addr)
}

// Remove session from group. Expects the global mutex to already be locked.
func (session *Session) removeFromGroup() {
	if session.groupPointer == nil {
		return
	}

	delete(session.groupPointer.players, session)

	if len(session.groupPointer.players) == 0 {
		logging.Notice("QR2", "Deleting group", aurora.Cyan(session.groupPointer.GroupName))
		delete(groups, session.groupPointer.GroupName)
	} else if session.groupPointer.server == session {
		logging.Notice("QR2", "Server down in group", aurora.Cyan(session.groupPointer.GroupName))
		session.groupPointer.server = nil
		session.groupPointer.findNewServer()
	}

	for player := range session.groupPointer.players {
		delete(player.Data, "+conn_"+session.Data["+joinindex"])
	}

	for field := range session.Data {
		if strings.HasPrefix(field, "+conn_") {
			delete(session.Data, field)
		}
	}

	session.groupPointer = nil
	session.GroupName = ""
}

// Update session data, creating the session if it doesn't exist. Returns a copy of the session data.
func setSessionData(moduleName string, addr net.Addr, sessionId uint32, payload map[string]string) (Session, bool) {
	newPID, newPIDValid := payload["dwc_pid"]
	delete(payload, "dwc_pid")

	lookupAddr := makeLookupAddr(addr.String())

	// Moving into performing operations on the session data, so lock the mutex
	mutex.Lock()
	defer mutex.Unlock()
	session, sessionExists := sessions[lookupAddr]

	if sessionExists && session.Addr.String() != addr.String() {
		logging.Error(moduleName, "Session IP mismatch")
		return Session{}, false
	}

	if !sessionExists {
		session = &Session{
			SessionID:       sessionId,
			Addr:            *addr.(*net.UDPAddr),
			Challenge:       "",
			Authenticated:   false,
			LastKeepAlive:   time.Now().UTC().Unix(),
			Endianness:      ClientNoEndian,
			Data:            payload,
			PacketCount:     0,
			Reservation:     common.MatchCommandData{},
			ReservationID:   0,
			messageMutex:    &deadlock.Mutex{},
			messageAckWaker: &sleep.Waker{},
		}
	}

	if newPIDValid && !session.setProfileID(moduleName, newPID, "") {
		return Session{}, false
	}

	if !sessionExists {
		logging.Info(moduleName, "Creating session", aurora.Cyan(sessionId).String())

		// Set search ID
		for {
			searchID := uint64(rand.Int63n((1<<24)-1) + 1)
			if _, exists := sessionBySearchID[searchID]; !exists {
				session.SearchID = searchID
				session.Data["+searchid"] = strconv.FormatUint(searchID, 10)
				sessionBySearchID[searchID] = session
				break
			}
		}

		sessions[lookupAddr] = session
		return *session, true
	}

	// Save certain fields
	for k, v := range session.Data {
		if k[0] == '+' || k == "dwc_pid" {
			payload[k] = v
		}
	}

	session.Data = payload
	session.LastKeepAlive = time.Now().UTC().Unix()
	session.SessionID = sessionId
	return *session, true
}

// Set the session's profile ID if it doesn't already exists.
// Returns false if the profile ID is invalid.
// Expects the global mutex to already be locked.
func (session *Session) setProfileID(moduleName string, newPID string, gpcmIP string) bool {
	if oldPID, oldPIDValid := session.Data["dwc_pid"]; oldPIDValid && oldPID != "" {
		if newPID != oldPID {
			logging.Error(moduleName, "New dwc_pid mismatch: new:", aurora.Cyan(newPID), "old:", aurora.Cyan(oldPID))
			return false
		}

		return true
	}

	// Setting a new PID so validate it
	profileID, err := strconv.ParseUint(newPID, 10, 32)
	if err != nil || strconv.FormatUint(profileID, 10) != newPID {
		logging.Error(moduleName, "Invalid dwc_pid value:", aurora.Cyan(newPID))
		return false
	}

	// Check if the public IP matches the one used for the GPCM session
	var gpPublicIP string
	var loginInfo *LoginInfo
	var ok bool
	if loginInfo, ok = logins[uint32(profileID)]; ok {
		gpPublicIP = strings.Split(loginInfo.GPPublicIP, ":")[0]
	} else {
		logging.Error(moduleName, "Provided dwc_pid is not logged in:", aurora.Cyan(newPID))
		return false
	}

	// TODO: Some kind of authentication
	if gpcmIP != "" && gpcmIP != gpPublicIP {
		logging.Error(moduleName, "TCP public IP mismatch: SB:", aurora.Cyan(gpcmIP), "GP:", aurora.Cyan(gpPublicIP))
		return false
	}

	if ratingError := checkValidRating(moduleName, session.Data); ratingError != "ok" {
		profileId := loginInfo.ProfileID

		mutex.Unlock()
		gpErrorCallback(profileId, ratingError)
		mutex.Lock()
		return false
	}

	session.login = loginInfo

	// Constraint: only one session can exist with a given profile ID
	if loginInfo.session != nil {
		logging.Notice(moduleName, "Removing outdated session", aurora.BrightCyan(loginInfo.session.Addr.String()), "with PID", aurora.Cyan(newPID))
		removeSession(makeLookupAddr(loginInfo.session.Addr.String()))
	}

	loginInfo.session = session

	if loginInfo.DeviceAuthenticated {
		session.Data["+deviceauth"] = "1"
	} else {
		session.Data["+deviceauth"] = "0"
	}

	session.Data["+gppublicip"], _ = common.IPFormatToString(gpPublicIP)
	session.Data["+fcgameid"] = loginInfo.FriendKeyGame

	session.Data["dwc_pid"] = newPID
	logging.Notice(moduleName, "Opened session with PID", aurora.Cyan(newPID))

	return true
}

func makeLookupAddr(addr string) uint64 {
	ip, port := common.IPFormatToInt(addr)
	return (uint64(port) << 32) | uint64(uint32(ip))
}

// Get a copy of the list of servers
func GetSessionServers() []map[string]string {
	var servers []map[string]string
	var unreachable []uint64
	currentTime := time.Now().UTC().Unix()

	mutex.Lock()
	defer mutex.Unlock()
	for sessionAddr, session := range sessions {
		// If the last keep alive was over a minute ago then consider the server unreachable
		if session.LastKeepAlive < currentTime-60 {
			// If the last keep alive was over an hour ago then remove the server
			if session.LastKeepAlive < currentTime-((60*60)*1) {
				unreachable = append(unreachable, sessionAddr)
			}
			continue
		}

		if !session.Authenticated {
			continue
		}

		servers = append(servers, session.Data)
	}

	// Remove unreachable sessions
	for _, sessionAddr := range unreachable {
		logging.Notice("QR2", "Removing unreachable session", aurora.BrightCyan(sessions[sessionAddr].Addr.String()))
		removeSession(sessionAddr)
	}

	return servers
}

func GetSearchID(addr uint64) uint64 {
	mutex.Lock()
	defer mutex.Unlock()

	if session := sessions[addr]; session != nil {
		return session.SearchID
	}

	return 0
}

// Save the sessions to a file. Expects the mutex to be locked.
func saveSessions() error {
	file, err := os.OpenFile("state/qr2_sessions.gob", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(sessions)
	file.Close()
	return err
}

// Load the sessions from a file. Expects the mutex to be locked.
func loadSessions() error {
	file, err := os.Open("state/qr2_sessions.gob")
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&sessions)
	file.Close()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.SearchID != 0 {
			sessionBySearchID[session.SearchID] = session
		}

		session.messageMutex = &deadlock.Mutex{}
		session.messageAckWaker = &sleep.Waker{}
		session.groupPointer = nil
		session.login = nil
	}

	return nil
}

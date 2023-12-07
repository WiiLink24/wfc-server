package qr2

import (
	"bytes"
	"encoding/binary"
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
	sessions = map[uint32]*Session{}
	mutex    = sync.RWMutex{}
)

// Remove a session.
func removeSession(sessionId uint32) {
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
			SearchID:      uint64(rand.Int63n(0x400 << 32)),
			Addr:          addr,
			Challenge:     "",
			Authenticated: false,
			LastKeepAlive: time.Now().Unix(),
			Endianness:    ClientNoEndian,
			Data:          payload,
			PacketCount:   0,
		}
		data.Data["+searchid"] = strconv.FormatUint(data.SearchID, 10)

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

func getSessionByPublicIP(publicIP uint32, publicPort uint16) *Session {
	ipStr := strconv.FormatInt(int64(int32(publicIP)), 10)
	portStr := strconv.FormatUint(uint64(publicPort), 10)

	currentTime := time.Now().Unix()
	mutex.Lock()
	defer mutex.Unlock()

	// Find the session with the IP
	for _, session := range sessions {
		if !session.Authenticated {
			continue
		}

		// If the last keep alive was over a minute ago then consider the server unreachable
		if session.LastKeepAlive < currentTime-60 {
			continue
		}

		if session.Data["publicip"] == ipStr && session.Data["publicport"] == portStr {
			return session
		}
	}

	return nil
}

func getSessionBySearchID(searchID uint64) *Session {
	currentTime := time.Now().Unix()
	mutex.Lock()
	defer mutex.Unlock()

	// Find the session with the ID
	for _, session := range sessions {
		if !session.Authenticated {
			continue
		}

		// If the last keep alive was over a minute ago then consider the server unreachable
		if session.LastKeepAlive < currentTime-60 {
			continue
		}

		if session.SearchID == searchID {
			return session
		}
	}

	return nil
}

func SendClientMessage(senderIP string, destSearchID uint64, message []byte) {
	moduleName := "QR2/MSG"

	var matchData common.MatchCommandData
	var sender *Session
	var receiver *Session
	senderIPInt, _ := common.IPFormatToInt(senderIP)

	useSearchID := destSearchID < (0x400 << 32)
	if useSearchID {
		receiver = getSessionBySearchID(destSearchID)
	} else {
		// It's an IP address, used in some circumstances
		receiverIPInt := uint32(destSearchID & 0xffffffff)
		receiverPort := uint16(destSearchID >> 32)
		receiver = getSessionByPublicIP(receiverIPInt, receiverPort)
	}

	if receiver == nil {
		logging.Error(moduleName, "Destination", aurora.Cyan(destSearchID), "does not exist")
		return
	}

	// Decode and validate the message
	isNatnegPacket := false
	if bytes.Equal(message[:2], []byte{0xfd, 0xfc}) {
		// Sending natneg cookie
		isNatnegPacket = true
		if len(message) != 0xA {
			logging.Error(moduleName, "Received invalid length NATNEG packet")
			return
		}

		senderSessionID := binary.LittleEndian.Uint32(message[0x6:0xA])
		moduleName = "QR2/MSG:s" + strconv.FormatUint(uint64(senderSessionID), 10)
	} else if bytes.Equal(message[:4], []byte{0xbb, 0x49, 0xcc, 0x4d}) {
		// DWC match command
		if len(message) < 0x14 {
			logging.Error(moduleName, "Received invalid length match command packet")
			return
		}

		senderProfileID := binary.LittleEndian.Uint32(message[0x10:0x14])
		moduleName = "QR2/MSG:p" + strconv.FormatUint(uint64(senderProfileID), 10)

		if (int(message[9]) + 0x14) != len(message) {
			logging.Error(moduleName, "Received invalid match command packet header")
			return
		}

		qr2IP := binary.BigEndian.Uint32(message[0x0C:0x10])
		qr2Port := binary.LittleEndian.Uint16(message[0x0A:0x0C])

		if senderIPInt != int32(qr2IP) {
			logging.Error(moduleName, "Wrong QR2 IP in match command packet header")
		}

		sender = getSessionByPublicIP(qr2IP, qr2Port)
		if sender == nil {
			logging.Error(moduleName, "Session does not exist with QR2 IP and port")
			return
		}

		var ok bool
		matchData, ok = common.DecodeMatchCommand(message[8], message[0x14:])
		if !ok {
			logging.Error(moduleName, "Received invalid match command:", aurora.Cyan(message[8]))
			return
		}

		if message[8] == common.MatchReservation {
			if matchData.Reservation.MatchType == 3 {
				// TODO: Check that this is correct
				logging.Error(moduleName, "RESERVATION: Attempt to join a private room over ServerBrowser")
				return
			}

			if qr2IP != matchData.Reservation.PublicIP {
				logging.Error(moduleName, "RESERVATION: Public IP mismatch in header and command")
				return
			}

			if matchData.Reservation.PublicPort < 1024 {
				logging.Error(moduleName, "RESERVATION: Public port is reserved")
				return
			}

			if matchData.Reservation.LocalPort < 1024 {
				logging.Error(moduleName, "RESERVATION: Local port is reserved")
				return
			}

			if useSearchID {
				matchData.Reservation.PublicIP = uint32(sender.SearchID & 0xffffffff)
				matchData.Reservation.PublicPort = uint16((sender.SearchID >> 32) & 0xffff)
				matchData.Reservation.LocalIP = 0
				matchData.Reservation.LocalPort = 0
			}
		}

		if message[8] == common.MatchResvOK {
			if qr2IP != matchData.ResvOK.PublicIP {
				logging.Error(moduleName, "RESERVATION: Public IP mismatch in header and command")
				return
			}

			if matchData.ResvOK.PublicPort < 1024 {
				logging.Error(moduleName, "RESV_OK: Public port is reserved")
				return
			}

			if matchData.ResvOK.LocalPort < 1024 {
				logging.Error(moduleName, "RESV_OK: Local port is reserved")
				return
			}

			if matchData.ResvOK.ProfileID != senderProfileID {
				logging.Error(moduleName, "RESV_OK: Profile ID mismatch in header")
				return
			}

			if useSearchID {
				matchData.ResvOK.PublicIP = uint32(sender.SearchID & 0xffffffff)
				matchData.ResvOK.PublicPort = uint16((sender.SearchID >> 32) & 0xffff)
				matchData.ResvOK.LocalIP = 0
				matchData.ResvOK.LocalPort = 0
			}
		}

		if message[8] == common.MatchTellAddr {
			mutex.Lock()
			if sender.Data["publicip"] != receiver.Data["publicip"] {
				mutex.Unlock()
				logging.Error(moduleName, "TELL_ADDR: Public IP does not match receiver")
				return
			}
			mutex.Unlock()

			if matchData.TellAddr.LocalPort < 1024 {
				logging.Error(moduleName, "TELL_ADDR: Local port is reserved")
				return
			}
		}

		if useSearchID {
			// Convert public IP to search ID
			qr2SearchID := binary.LittleEndian.AppendUint16([]byte{}, uint16((sender.SearchID>>32)&0xffff))
			qr2SearchID = binary.BigEndian.AppendUint32(qr2SearchID, uint32(sender.SearchID&0xffffffff))
			message = append(message[:0x0A], append(qr2SearchID, message[0x10:0x14]...)...)
		} else {
			message = message[:0x14]
		}

		var matchMessage []byte
		matchMessage, ok = common.EncodeMatchCommand(message[8], matchData)
		if !ok {
			logging.Error(moduleName, "Failed to reencode match command:", aurora.Cyan(message[8]))
			return
		}

		if len(matchMessage) != 0 {
			message = append(message, matchMessage...)
		}
	}

	mutex.Lock()

	destPid, ok := receiver.Data["dwc_pid"]
	if !ok || destPid == "" {
		destPid = "<UNKNOWN>"
	}

	destSessionID := receiver.SessionID
	packetCount := receiver.PacketCount + 1
	receiver.PacketCount = packetCount
	destAddr := receiver.Addr

	mutex.Unlock()

	if isNatnegPacket {
		cookie := binary.BigEndian.Uint32(message[0x2:0x6])
		logging.Notice(moduleName, "Send NN cookie", aurora.Cyan(cookie), "to", aurora.BrightCyan(destPid))
	} else {
		common.LogMatchCommand(moduleName, destPid, message[8], matchData)
	}

	payload := createResponseHeader(ClientMessageRequest, destSessionID)

	payload = append(payload, []byte{0, 0, 0, 0}...)
	binary.BigEndian.PutUint32(payload[len(payload)-4:], packetCount)
	payload = append(payload, message...)

	// TODO: Send again if no CLIENT_MESSAGE_ACK is received after
	_, err := masterConn.WriteTo(payload, destAddr)
	if err != nil {
		logging.Error(moduleName, "Error sending message:", err.Error())
	}
}

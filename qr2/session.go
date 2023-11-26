package qr2

import (
	"bytes"
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
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
	Addr          net.Addr
	Challenge     string
	Authenticated bool
	LastKeepAlive int64
	Endianness    byte // Some fields depend on the client's endianness
	Data          map[string]string
	PacketCount   uint32
}

// Remove a session.
func removeSession(sessionId uint32) {
	delete(sessions, sessionId)
}

// Update session data, creating the session if it doesn't exist. Returns a copy of the session data.
func setSessionData(sessionId uint32, payload map[string]string) (Session, bool) {
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
			Challenge:     "",
			Authenticated: false,
			LastKeepAlive: time.Now().Unix(),
			Endianness:    ClientNoEndian,
			Data:          payload,
			PacketCount:   0,
		}
		sessions[sessionId] = &data

		return data, true
	}

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

func SendClientMessage(destIP string, message []byte) {
	moduleName := "QR2/MSG"

	var matchData common.MatchCommandData
	senderProfileID := uint32(0)

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

		senderProfileID = binary.LittleEndian.Uint32(message[0x10:0x14])
		moduleName = "QR2/MSG:p" + strconv.FormatUint(uint64(senderProfileID), 10)

		if (int(message[9]) + 0x14) != len(message) {
			logging.Error(moduleName, "Received invalid match command packet header")
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

			if common.IsReservedIP(int32(matchData.Reservation.PublicIP)) {
				logging.Warn(moduleName, "RESERVATION: Public IP is reserved")
				// Temporarily disabled for localhost testing
				// TODO: Add a config option or something
			}

			if matchData.Reservation.PublicPort < 1024 {
				logging.Error(moduleName, "RESERVATION: Public port is reserved")
				return
			}

			if matchData.Reservation.LocalPort < 1024 {
				logging.Error(moduleName, "RESERVATION: Local port is reserved")
				return
			}
		}

		if message[8] == common.MatchResvOK {
			if common.IsReservedIP(int32(matchData.ResvOK.PublicIP)) {
				logging.Warn(moduleName, "RESV_OK: Public IP is reserved")
				// TODO: See above
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
		}

		if message[8] == common.MatchTellAddr {
			// TODO: Check if the public IPs are actually the same
			if matchData.TellAddr.LocalPort < 1024 {
				logging.Error(moduleName, "TELL_ADDR: Local port is reserved")
				return
			}
		}
	}

	destIPIntStr, destPortStr := common.IPFormatToString(destIP)

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

		if session.Data["publicip"] == destIPIntStr && session.Data["publicport"] == destPortStr {
			destPid, ok := session.Data["dwc_pid"]
			if !ok || destPid == "" {
				break
			}

			destSessionID := session.SessionID
			packetCount := session.PacketCount + 1
			session.PacketCount = packetCount

			mutex.Unlock()

			if isNatnegPacket {
				cookie := binary.BigEndian.Uint32(message[0x2:0x6])
				logging.Notice(moduleName, "Send NN cookie", aurora.Cyan(cookie), "to", aurora.BrightCyan(destPid))
			} else {
				common.LogMatchCommand(moduleName, destPid, message[8], matchData)
			}

			// Found the client, now send the message
			payload := createResponseHeader(ClientMessageRequest, destSessionID)

			payload = append(payload, []byte{0, 0, 0, 0}...)
			binary.BigEndian.PutUint32(payload[len(payload)-4:], packetCount)
			payload = append(payload, message...)

			destIPAddr, err := net.ResolveUDPAddr("udp", destIP)
			if err != nil {
				panic(err)
			}

			// TODO: Send again if no CLIENT_MESSAGE_ACK is received after
			masterConn.WriteTo(payload, destIPAddr)
			return
		}
	}
	mutex.Unlock()

	logging.Error(moduleName, "Could not find destination server")
}

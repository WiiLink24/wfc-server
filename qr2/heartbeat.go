package qr2

import (
	"encoding/binary"
	"net"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func heartbeat(moduleName string, conn net.PacketConn, addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	logging.Info(moduleName, "Received heartbeat; session ID:", aurora.BrightCyan(sessionId))

	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	unknowns := []string{}
	for i := 0; i < len(values); i += 2 {
		if len(values[i]) == 0 || values[i][0] == '+' {
			continue
		}

		logging.Info(moduleName, aurora.Cyan(values[i]).String()+":", aurora.Cyan(values[i+1]))

		if values[i] == "unknown" {
			unknowns = append(unknowns, values[i+1])
			continue
		}

		payload[values[i]] = values[i+1]
	}

	realIP, realPort := common.IPFormatToString(addr.String())

	if ip, ok := payload["publicip"]; !ok || ip == "0" {
		// Set the public IP key to the real IP
		payload["publicip"] = realIP
		payload["publicport"] = realPort
	}

	// Client is mistaken about its public IP
	if payload["publicip"] != realIP || payload["publicport"] != realPort {
		logging.Error(moduleName, "Public IP mismatch")
		return
	}

	lookupAddr := makeLookupAddr(addr.String())

	statechanged, ok := payload["statechanged"]
	if ok {
		if statechanged == "1" {
			// TODO: This would be a good place to run the server->client message exploit
			// for DNS patcher games that require code patches. The status code should be
			// set to 5 at this point (if publicip is not 0), which is required.
			logging.Notice(moduleName, "Client session update")
			// Fall through
		}

		if statechanged == "2" {
			logging.Notice(moduleName, "Client session shutdown")
			mutex.Lock()
			removeSession(lookupAddr)
			mutex.Unlock()
			return
		}
	}

	session, ok := setSessionData(moduleName, addr, sessionId, payload)
	if !ok {
		return
	}

	if payload["gamename"] == "mariokartwii" && len(unknowns) > 0 {
		// Try to login using the first unknown as a profile ID
		// This makes it possible to execute the exploit on the client sooner
		profileId := unknowns[0]
		logging.Notice(moduleName, "Attempting to use unknown as profile ID", aurora.Cyan(profileId))

		mutex.Lock()
		session, sessionExists := sessions[lookupAddr]
		if !sessionExists {
			logging.Error(moduleName, "Session not found")
		} else {
			session.setProfileID(moduleName, profileId)
		}
		mutex.Unlock()
	}

	if !session.Authenticated {
		logging.Notice(moduleName, "Sending challenge")
		sendChallenge(conn, addr, session, lookupAddr)
	} else if !session.ExploitReceived && session.Login != nil && session.Login.NeedsExploit && statechanged == "1" {
		logging.Notice(moduleName, "Sending SBCM exploit to DNS patcher client")
		sendClientExploit(moduleName, session)
	}

	mutex.Lock()
	if session.GroupPointer != nil {
		if session.GroupPointer.Server == nil {
			session.GroupPointer.findNewServer()
		} else {
			// Update the match type if needed
			session.GroupPointer.updateMatchType()
		}
	}
	mutex.Unlock()

	logging.Info(moduleName, "Heartbeat ok")
}

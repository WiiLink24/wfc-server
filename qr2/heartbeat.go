package qr2

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func heartbeat(moduleName string, conn net.PacketConn, addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	unknowns := []string{}
	for i := 0; i < len(values); i += 2 {
		if len(values[i]) == 0 || values[i][0] == '+' {
			continue
		}

		if values[i] == "unknown" {
			unknowns = append(unknowns, values[i+1])
			continue
		}

		payload[values[i]] = values[i+1]
	}

	if payload["dwc_mtype"] != "" {
		logging.Info(moduleName, "Match type:", aurora.Cyan(payload["dwc_mtype"]))
	}

	if payload["dwc_hoststate"] != "" {
		logging.Info(moduleName, "Host state:", aurora.Cyan(payload["dwc_hoststate"]))
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
	if ok && statechanged == "2" {
		logging.Notice(moduleName, "Client session shutdown")
		mutex.Lock()
		removeSession(lookupAddr)
		mutex.Unlock()
		return
	}

	if ratingError := checkValidRating(moduleName, payload); ratingError != "ok" {
		mutex.Lock()
		session, sessionExists := sessions[lookupAddr]
		if sessionExists && session.Login != nil {
			callback := session.Login.GPErrorCallback
			profileId := session.Login.ProfileID

			mutex.Unlock()
			callback(profileId, ratingError)
			return
		} else {
			// Else don't return and move on, so we can return an error once logged in
			mutex.Unlock()
		}
	}

	session, ok := setSessionData(moduleName, addr, sessionId, payload)
	if !ok {
		return
	}

	if payload["gamename"] == "mariokartwii" && len(unknowns) > 0 {
		// Try to login using the first unknown as a profile ID
		// This makes it possible to execute the exploit on the client sooner

		mutex.Lock()
		session, sessionExists := sessions[lookupAddr]
		if !sessionExists {
			logging.Error(moduleName, "Session not found")
		} else if session.Login == nil {
			profileId := unknowns[0]
			logging.Info(moduleName, "Attempting to use unknown as profile ID", aurora.Cyan(profileId))
			session.setProfileID(moduleName, profileId)
		}
		mutex.Unlock()
	}

	if !session.Authenticated {
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
}

func checkValidRating(moduleName string, payload map[string]string) string {
	if payload["gamename"] == "mariokartwii" {
		// ev and eb values must be in range 1 to 9999
		if ev := payload["ev"]; ev != "" {
			evInt, err := strconv.ParseInt(ev, 10, 16)
			if err != nil || evInt < 1 || evInt > 9999 {
				logging.Error(moduleName, "Invalid ev value:", aurora.Cyan(ev))
				return "invalid_elo"
			}
		}

		if eb := payload["eb"]; eb != "" {
			ebInt, err := strconv.ParseInt(eb, 10, 16)
			if err != nil || ebInt < 1 || ebInt > 9999 {
				logging.Error(moduleName, "Invalid eb value:", aurora.Cyan(eb))
				return "invalid_elo"
			}
		}
	}

	return "ok"
}

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

func heartbeat(moduleName string, conn net.PacketConn, addr net.UDPAddr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	unknowns := []string{}
	for i := 0; i < len(values)-1; i += 2 {
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

	noIP := false
	if ip, ok := payload["publicip"]; !ok || ip == "0" {
		noIP = true
	}

	clientEndianness := common.GetExpectedUnitCode(payload["gamename"])
	if !noIP && clientEndianness == ClientBigEndian {
		if payload["publicip"] != realIP || payload["publicport"] != realPort {
			// Client is mistaken about its public IP
			logging.Error(moduleName, "Public IP mismatch")
			return
		}
	} else if !noIP && clientEndianness == ClientLittleEndian {
		realIPLE, realPortLE := common.IPFormatToStringLE(addr.String())
		if payload["publicip"] != realIPLE || payload["publicport"] != realPortLE {
			// Client is mistaken about its public IP
			logging.Error(moduleName, "Public IP mismatch")
			return
		}
	}
	// Else it's a cross-compatible game and the endianness is ambiguous

	payload["publicip"] = realIP
	payload["publicport"] = realPort

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
		if sessionExists && session.login != nil {
			profileId := session.login.ProfileID

			mutex.Unlock()
			gpErrorCallback(profileId, ratingError)
			return
		} else {
			// Else don't return and move on, so we can return an error once logged in
			mutex.Unlock()
		}
	}

	session, ok := setSessionData(moduleName, &addr, sessionId, payload)
	if !ok {
		return
	}

	if len(unknowns) > 0 {
		// Try to login using the first unknown as a profile ID
		// This makes it possible to execute the exploit on the client sooner

		mutex.Lock()
		sessionPtr, sessionExists := sessions[lookupAddr]
		if !sessionExists {
			logging.Error(moduleName, "Session not found")
		} else if sessionPtr.login == nil {
			profileId := unknowns[0]
			logging.Info(moduleName, "Attempting to use unknown as profile ID", aurora.Cyan(profileId))
			sessionPtr.setProfileID(moduleName, profileId, "")
		}
		session = *sessionPtr
		mutex.Unlock()
	}

	if !session.Authenticated || noIP {
		sendChallenge(conn, addr, session, lookupAddr)
	}

	if login := session.login; !session.ExploitReceived && login != nil && session.login.NeedsExploit {
		// The version of DWC in Mario Kart DS doesn't check matching status
		if (!noIP && statechanged == "1") || login.GameCode == "AMCE" || login.GameCode == "AMCP" || login.GameCode == "AMCJ" {
			logging.Notice(moduleName, "Sending SBCM exploit to DNS patcher client")
			sendClientExploit(moduleName, session)
		}
	}

	mutex.Lock()
	if session.groupPointer != nil {
		if session.groupPointer.server == nil {
			session.groupPointer.findNewServer()
		} else {
			// Update the match type if needed
			session.groupPointer.updateMatchType()
		}
	}
	mutex.Unlock()
}

func checkValidRating(moduleName string, payload map[string]string) string {
	if payload["gamename"] != "mariokartwii" {
		return "ok"
	}

	if public, isBattle := isPublicMatchRegion(payload["rk"]); public {
		// ev and eb values must be in range 1 to 9999
		if ev := payload["ev"]; !isBattle && ev != "" {
			evInt, err := strconv.ParseInt(ev, 10, 16)
			if err != nil || evInt < 1 || evInt > 9999 {
				logging.Error(moduleName, "Invalid ev value:", aurora.Cyan(ev))
				return "invalid_elo"
			}
		} else if eb := payload["eb"]; isBattle && eb != "" {
			ebInt, err := strconv.ParseInt(eb, 10, 16)
			if err != nil || ebInt < 1 || ebInt > 9999 {
				logging.Error(moduleName, "Invalid eb value:", aurora.Cyan(eb))
				return "invalid_elo"
			}
		}
	}
	return "ok"
}

func isPublicMatchRegion(rk string) (bool, bool) {
	if rk == "vs" {
		return true, false
	} else if rk == "bt" {
		return true, true
	} else if len(rk) == 4 && rk[3] >= '0' && rk[3] < '6' {
		if strings.HasPrefix(rk, "vs_") {
			return true, false
		} else if strings.HasPrefix(rk, "bt_") {
			return true, true
		}
	}
	return false, false
}

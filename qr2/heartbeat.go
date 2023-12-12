package qr2

import (
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

func heartbeat(moduleName string, conn net.PacketConn, addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	logging.Info(moduleName, "Received heartbeat; session ID:", aurora.BrightCyan(sessionId))

	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	for i := 0; i < len(values); i += 2 {
		if len(values[i]) == 0 || values[i][0] == '+' {
			continue
		}

		payload[values[i]] = values[i+1]
		logging.Info(moduleName, aurora.Cyan(values[i]).String()+":", aurora.Cyan(values[i+1]))
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

	if statechanged, ok := payload["statechanged"]; ok {
		if statechanged == "1" {
			// TODO: This would be a good place to run the server->client message exploit
			// for DNS patcher games that require code patches. The status code should be
			// set to 5 at this point (if publicip is not 0), which is required.
			logging.Notice(moduleName, "Client session update")
			// Fall through
		}

		if statechanged == "2" {
			logging.Notice(moduleName, "Client session shutdown")
			removeSession(lookupAddr)
			return
		}
	}

	session, ok := setSessionData(moduleName, addr, sessionId, payload)
	if !ok {
		return
	}

	if !session.Authenticated {
		logging.Notice(moduleName, "Sending challenge")
		sendChallenge(conn, addr, session, lookupAddr)
		return
	}
}

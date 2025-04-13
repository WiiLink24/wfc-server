package gpcm

import (
	"strings"
	"time"
	"wwfc/common"
	"wwfc/qr2"
)

func kickPlayer(profileID uint32, reason string) {
	pids := []uint32{profileID}

	if session, exists := sessions[profileID]; exists {
		errorMessage := WWFCMsgKickedGeneric

		switch reason {
		case "banned":
			errorMessage = WWFCMsgProfileBannedTOSNow

		case "restricted":
			errorMessage = WWFCMsgProfileRestrictedNow

		case "restricted_join":
			errorMessage = WWFCMsgProfileRestricted

		case "moderator_kick":
			errorMessage = WWFCMsgKickedModerator

		case "room_kick":
			errorMessage = WWFCMsgKickedRoomHost

		case "invalid_elo":
			errorMessage = WWFCMsgInvalidELO

		case "network_error":
			// No error message
			common.CloseConnection(ServerName, session.ConnIndex)
			return
		}

		gpError := GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the server. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: errorMessage,
		}

		for _, match := range findMatchingSessions(session) {
			pids = append(pids, match.User.ProfileId)
			match.replyError(gpError)
		}

		session.replyError(gpError)
	}

	// After 3 seconds, send kick order to all players
	// This is to prevent the restricted player from staying in the group if he ignores the GPCM kick
	go func() {
		time.AfterFunc(3*time.Second, func() {
			qr2.OrderKickFromGroups(pids)
		})
	}()
}

func KickPlayer(profileID uint32, reason string) {
	mutex.Lock()
	defer mutex.Unlock()

	kickPlayer(profileID, reason)
}

func findMatchingSessions(badSession *GameSpySession) []*GameSpySession {
	ret := []*GameSpySession{}

	badAddrSplit := strings.Split(badSession.RemoteAddr, ":")

	var badAddr string
	// If the bad address cannot be split for some reason just send a blank string
	if len(badAddrSplit) > 0 {
		badAddr = badAddrSplit[0]
	} else {
		badAddr = ""
	}

	for _, session := range sessions {
		if session.ConnIndex == badSession.ConnIndex {
			// We already know to kick this one. Don't try and kick twice.
			continue
		}

		if badSession.DeviceId != 67349608 && badSession.DeviceId == session.DeviceId {
			ret = append(ret, session)
			continue
		}

		if badAddr == "" {
			continue
		}

		// Addresses are in the form of IP:Port
		addrSplit := strings.Split(session.RemoteAddr, ":")

		if len(addrSplit) == 0 {
			continue
		}

		addr := addrSplit[0]

		if addr == badAddr {
			ret = append(ret, session)
		}
	}

	return ret
}

func KickPlayerCustomMessage(profileID uint32, reason string, message WWFCErrorMessage) {
	mutex.Lock()
	defer mutex.Unlock()

	pids := []uint32{profileID}

	if session, exists := sessions[profileID]; exists {
		gpError := GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the server. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: message,
			Reason:      reason,
		}

		for _, match := range findMatchingSessions(session) {
			pids = append(pids, match.User.ProfileId)
			match.replyError(gpError)
		}

		session.replyError(gpError)
	}

	// After 3 seconds, send kick order to all players
	// This is to prevent the restricted player from staying in the group if he ignores the GPCM kick
	go func() {
		time.AfterFunc(3*time.Second, func() {
			qr2.OrderKickFromGroups(pids)
		})
	}()
}

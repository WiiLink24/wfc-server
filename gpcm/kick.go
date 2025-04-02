package gpcm

import (
	"time"
	"wwfc/common"
	"wwfc/qr2"
)

func kickPlayer(profileID uint32, reason string) {
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

		session.replyError(GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the server. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: errorMessage,
		})
	}

	// After 3 seconds, send kick order to all players
	// This is to prevent the restricted player from staying in the group if he ignores the GPCM kick
	go func() {
		time.AfterFunc(3*time.Second, func() {
			qr2.OrderKickFromGroups(profileID)
		})
	}()
}

func KickPlayer(profileID uint32, reason string) {
	mutex.Lock()
	defer mutex.Unlock()

	kickPlayer(profileID, reason)
}

func KickPlayerCustomMessage(profileID uint32, reason string, message WWFCErrorMessage) {
	mutex.Lock()
	defer mutex.Unlock()

	if session, exists := sessions[profileID]; exists {
		session.replyError(GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the server. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: message,
			Reason:      reason,
		})
	}

	// After 3 seconds, send kick order to all players
	// This is to prevent the restricted player from staying in the group if he ignores the GPCM kick
	go func() {
		time.AfterFunc(3*time.Second, func() {
			qr2.OrderKickFromGroups(profileID)
		})
	}()
}

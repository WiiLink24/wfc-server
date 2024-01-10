package gpcm

func KickPlayer(profileID uint32, reason string) {
	mutex.Lock()
	defer mutex.Unlock()

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
		}

		session.replyError(GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the game. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: errorMessage,
		})
		session.Conn.Close()
	}
}

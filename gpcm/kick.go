package gpcm

import (
	"strconv"
	"time"
	"wwfc/common"
	"wwfc/qr2"
)

// Expects a global mutex to be locked
// This function will return all sessions that are matched with the player `profileID`
func getSessionsMatchedWithPlayer(gameName string, profileID uint32) []*GameSpySession {

	var matchedPlayers []*GameSpySession

	// Check if the user is part of a group
	var foundGroup *qr2.GroupInfo
	groups := qr2.GetGroups([]string{gameName}, []string{}, false)
	for _, group := range groups {
		for _, playerInfo := range group.Players {
			pid, err := strconv.ParseUint(playerInfo.ProfileID, 10, 32)
			if err == nil {
				if uint32(pid) == profileID {
					foundGroup = &group
					break
				}
			}
		}
	}

	// If the user is part of a group, send a kick order to all players in the group
	if foundGroup != nil {
		for _, playerInfo := range foundGroup.Players {
			pid, err := strconv.ParseUint(playerInfo.ProfileID, 10, 32)
			if err == nil {
				if session, exists := sessions[uint32(pid)]; exists {
					matchedPlayers = append(matchedPlayers, session)
				}
			}
		}
	}

	return matchedPlayers
}

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

		players := getSessionsMatchedWithPlayer(session.GameName, profileID)
		session.replyError(GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the server. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: errorMessage,
		})

		// After 3 seconds, send kick order to all players in the group
		// This is to prevent the restricted player from staying in the group if he ignores the GPCM kick
		go func(players []*GameSpySession) {
			time.AfterFunc(3*time.Second, func() {
				for _, player := range players {
					qr2.OrderKickFromGroup(player.User.ProfileId, profileID)
				}
			})
		}(players)
	}
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
		players := getSessionsMatchedWithPlayer(session.GameName, profileID)
		session.replyError(GPError{
			ErrorCode:   ErrConnectionClosed.ErrorCode,
			ErrorString: "The player was kicked from the server. Reason: " + reason,
			Fatal:       true,
			WWFCMessage: message,
			Reason:      reason,
		})

		// After 3 seconds, send kick order to all players in the group
		// This is to prevent the restricted player from staying in the group if he ignores the GPCM kick
		go func(players []*GameSpySession) {
			time.AfterFunc(3*time.Second, func() {
				for _, player := range players {
					qr2.OrderKickFromGroup(player.User.ProfileId, profileID)
				}
			})
		}(players)
	}
}

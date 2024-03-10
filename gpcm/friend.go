package gpcm

import (
	"errors"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
)

func removeFromUint32Array(arrayPointer *[]uint32, index int) error {
	array := *arrayPointer
	arrayLength := len(array)

	if index < 0 || index >= arrayLength {
		return errors.New("index is out of bounds")
	}

	lastIndex := arrayLength - 1

	array[index] = array[lastIndex]
	*arrayPointer = array[:lastIndex]
	return nil
}

func (g *GameSpySession) isFriendAdded(profileId uint32) bool {
	for _, storedPid := range g.FriendList {
		if storedPid == profileId {
			return true
		}
	}
	return false
}

func (g *GameSpySession) getFriendIndex(profileId uint32) int {
	for i, storedPid := range g.FriendList {
		if storedPid == profileId {
			return i
		}
	}
	return -1
}

func (g *GameSpySession) isFriendAuthorized(profileId uint32) bool {
	for _, storedPid := range g.AuthFriendList {
		if storedPid == profileId {
			return true
		}
	}
	return false
}

func (g *GameSpySession) getAuthorizedFriendIndex(profileId uint32) int {
	for i, storedPid := range g.AuthFriendList {
		if storedPid == profileId {
			return i
		}
	}
	return -1
}

const (
	// addFriendMessage = "\r\n\r\n|signed|00000000000000000000000000000000"

	// TODO: Check if this is needed for any game, it sends via bm 1
	// authFriendMessage = "I have authorized your request to add me to your list"

	logOutMessage = "|s|0|ss|Offline|ls||ip|0|p|0|qm|0"
)

func (g *GameSpySession) addFriend(command common.GameSpyCommand) {
	strNewProfileId := command.OtherValues["newprofileid"]
	newProfileId, err := strconv.ParseUint(strNewProfileId, 10, 32)
	if err != nil {
		g.replyError(ErrAddFriend)
		return
	}

	if newProfileId == uint64(g.User.ProfileId) {
		logging.Error(g.ModuleName, "Attempt to add self as friend")
		g.replyError(ErrAddFriendBadNew)
		return
	}

	fc := common.CalcFriendCodeString(uint32(newProfileId), g.User.GsbrCode[:4])
	logging.Info(g.ModuleName, "Add friend:", aurora.Cyan(strNewProfileId), aurora.Cyan(fc))

	if g.isFriendAuthorized(uint32(newProfileId)) {
		logging.Info(g.ModuleName, "Attempt to add a friend who is already authorized")
		// This seems to always happen, do we need to return an error?
		// DWC vocally ignores the error anyway, so let's not bother
		// g.replyError(ErrAddFriendAlreadyFriends)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// TODO: Add a limit
	if !g.isFriendAdded(uint32(newProfileId)) {
		g.FriendList = append(g.FriendList, uint32(newProfileId))
	}

	// Check if destination has added the sender
	newSession, ok := sessions[uint32(newProfileId)]
	if !ok || newSession == nil || !newSession.LoggedIn {
		logging.Info(g.ModuleName, "Destination is not online")
		return
	}

	if newSession.GameName != g.GameName {
		logging.Error(g.ModuleName, "Destination is not playing the same game")
		// g.replyError(ErrAddFriendBadNew)
		return
	}

	if !newSession.User.OpenHost && !newSession.isFriendAdded(g.User.ProfileId) {
		// Not an error, just ignore for now
		logging.Info(g.ModuleName, "Destination has not added sender")
		return
	}

	// Friends are now mutual!
	// TODO: Add a limit
	g.AuthFriendList = append(g.AuthFriendList, uint32(newProfileId))
	newSession.AuthFriendList = append(newSession.AuthFriendList, g.User.ProfileId)

	// Send friend auth message
	sendMessageToSessionBuffer("4", newSession.User.ProfileId, g, "")
	if newSession.isFriendAdded(g.User.ProfileId) {
		sendMessageToSession("4", g.User.ProfileId, newSession, "")
	}

	g.exchangeFriendStatus(uint32(newProfileId))
}

func (g *GameSpySession) removeFriend(command common.GameSpyCommand) {
	strDelProfileID := command.OtherValues["delprofileid"]
	delProfileID64, err := strconv.ParseUint(strDelProfileID, 10, 32)
	if err != nil {
		logging.Error(g.ModuleName, aurora.Cyan(strDelProfileID), "is not a valid profile id")
		g.replyError(ErrDeleteFriend)
		return
	}
	delProfileID32 := uint32(delProfileID64)

	mutex.Lock()
	defer mutex.Unlock()

	if g.isFriendAdded(delProfileID32) {
		delProfileIDIndex := g.getFriendIndex(delProfileID32)
		removeFromUint32Array(&g.FriendList, delProfileIDIndex)
	}

	if !g.User.OpenHost {
		if g.isFriendAuthorized(delProfileID32) {
			delProfileIDIndex := g.getAuthorizedFriendIndex(delProfileID32)
			removeFromUint32Array(&g.AuthFriendList, delProfileIDIndex)
		}

		if session, ok := sessions[delProfileID32]; ok && session.LoggedIn && session.isFriendAuthorized(g.User.ProfileId) {
			sendMessageToSession("100", g.User.ProfileId, session, logOutMessage)
		}
	}
}

func (g *GameSpySession) authAddFriend(command common.GameSpyCommand) {
	strFromProfileId := command.OtherValues["fromprofileid"]
	fromProfileId, err := strconv.ParseUint(strFromProfileId, 10, 32)
	if err != nil {
		logging.Error(g.ModuleName, "Invalid profile ID string:", aurora.Cyan(strFromProfileId))
		g.replyError(ErrAuthAddBadFrom)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if !g.isFriendAuthorized(uint32(fromProfileId)) {
		logging.Error(g.ModuleName, "Sender", aurora.Cyan(fromProfileId), "is not an authorized friend")
		g.replyError(ErrAuthAddBadFrom)
		return
	}

	g.exchangeFriendStatus(uint32(fromProfileId))
}

func (g *GameSpySession) setStatus(command common.GameSpyCommand) {
	status := command.CommandValue
	logging.Notice(g.ModuleName, "New status:", aurora.BrightMagenta(status))

	qr2.ProcessGPStatusUpdate(g.User.ProfileId, g.QR2IP, status)

	statstring, ok := command.OtherValues["statstring"]
	if !ok {
		logging.Warn(g.ModuleName, "Missing statstring")
		statstring = ""
	}

	locstring, ok := command.OtherValues["locstring"]
	if !ok {
		logging.Warn(g.ModuleName, "Missing locstring")
		locstring = ""
	}

	statusMsg := "|s|" + status + "|ss|" + statstring + "|ls|" + locstring + "|ip|0|p|0|qm|0"

	mutex.Lock()
	defer mutex.Unlock()

	if status == "3" && g.User.Restricted {
		logging.Warn(g.ModuleName, "Restricted user searching for public rooms")
		kickPlayer(g.User.ProfileId, "restricted_join")
	}

	g.LocString = locstring
	g.Status = statusMsg

	for _, storedPid := range g.FriendList {
		g.sendFriendStatus(storedPid)
	}
}

func sendMessageToSession(msgType string, from uint32, session *GameSpySession, msg string) {
	message := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "bm",
		CommandValue: msgType,
		OtherValues: map[string]string{
			"f":   strconv.FormatUint(uint64(from), 10),
			"msg": msg,
		},
	})
	session.Conn.Write([]byte(message))
}

func sendMessageToSessionBuffer(msgType string, from uint32, session *GameSpySession, msg string) {
	session.WriteBuffer += common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "bm",
		CommandValue: msgType,
		OtherValues: map[string]string{
			"f":   strconv.FormatUint(uint64(from), 10),
			"msg": msg,
		},
	})
}

func sendMessageToProfileId(msgType string, from uint32, to uint32, msg string) bool {
	if session, ok := sessions[to]; ok && session.LoggedIn {
		sendMessageToSession(msgType, from, session, msg)
		return true
	}

	logging.Info("GPCM", "Destination", aurora.Cyan(to), "from", aurora.Cyan(from), "is not online")
	return false
}

func (g *GameSpySession) sendFriendStatus(profileId uint32) {
	common.UNUSED(sendMessageToProfileId)

	if !g.isFriendAdded(profileId) {
		return
	}

	if session, ok := sessions[profileId]; ok && session.LoggedIn && session.isFriendAdded(g.User.ProfileId) {
		// Prevent players abusing a stack overflow exploit with the locstring in Mario Kart Wii
		if session.NeedsExploit && strings.HasPrefix(session.GameCode, "RMC") && len(g.LocString) > 0x14 {
			logging.Warn("GPCM", "Blocked message from", aurora.Cyan(g.User.ProfileId), "to", aurora.Cyan(session.User.ProfileId), "due to a stack overflow exploit")
			return
		}

		sendMessageToSession("100", g.User.ProfileId, session, g.Status)
	}
}

func (g *GameSpySession) exchangeFriendStatus(profileId uint32) {
	if !g.isFriendAdded(profileId) {
		return
	}

	if session, ok := sessions[profileId]; ok && session.LoggedIn {
		if session.isFriendAdded(g.User.ProfileId) {
			if session.NeedsExploit && strings.HasPrefix(session.GameCode, "RMC") && len(g.LocString) > 0x14 {
				logging.Warn("GPCM", "Blocked message from", aurora.Cyan(g.User.ProfileId), "to", aurora.Cyan(session.User.ProfileId), "due to a stack overflow exploit")
				return
			}

			sendMessageToSession("100", g.User.ProfileId, session, g.Status)
		}

		if g.NeedsExploit && strings.HasPrefix(g.GameCode, "RMC") && len(session.LocString) > 0x14 {
			logging.Warn("GPCM", "Blocked message from", aurora.Cyan(session.User.ProfileId), "to", aurora.Cyan(g.User.ProfileId), "due to a stack overflow exploit")
			return
		}

		sendMessageToSessionBuffer("100", profileId, g, session.Status)
	}
}

func (g *GameSpySession) sendLogoutStatus() {
	mutex.Lock()
	defer mutex.Unlock()

	for _, storedPid := range g.AuthFriendList {
		if session, ok := sessions[storedPid]; ok && session.LoggedIn && session.isFriendAuthorized(g.User.ProfileId) {
			delProfileIDIndex := session.getAuthorizedFriendIndex(g.User.ProfileId)
			removeFromUint32Array(&session.AuthFriendList, delProfileIDIndex)
			sendMessageToSession("100", g.User.ProfileId, session, logOutMessage)
		}
	}
}

func (g *GameSpySession) openHostEnabled() {
	mutex.Lock()
	defer mutex.Unlock()

	for _, session := range sessions {
		if session.LoggedIn && session.isFriendAdded(g.User.ProfileId) && !session.isFriendAuthorized(g.User.ProfileId) {
			session.AuthFriendList = append(session.AuthFriendList, g.User.ProfileId)
			g.AuthFriendList = append(g.AuthFriendList, session.User.ProfileId)
			sendMessageToSession("4", g.User.ProfileId, session, "")
			session.exchangeFriendStatus(g.User.ProfileId)
		}
	}
}

func (g *GameSpySession) openHostDisabled() {
	mutex.Lock()
	defer mutex.Unlock()

	for _, id := range g.AuthFriendList {
		if g.isFriendAdded(id) {
			return
		}

		delProfileIDIndex := g.getAuthorizedFriendIndex(id)
		removeFromUint32Array(&g.AuthFriendList, delProfileIDIndex)

		if session, ok := sessions[id]; ok && session.LoggedIn && session.isFriendAuthorized(g.User.ProfileId) {
			delProfileIDIndex := session.getAuthorizedFriendIndex(g.User.ProfileId)
			removeFromUint32Array(&session.AuthFriendList, delProfileIDIndex)
			sendMessageToSession("100", g.User.ProfileId, session, logOutMessage)
		}
	}
}

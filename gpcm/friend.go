package gpcm

import (
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

func (g *GameSpySession) isFriendAdded(profileId uint32) bool {
	for _, storedPid := range g.FriendList {
		if storedPid == profileId {
			return true
		}
	}
	return false
}

func (g *GameSpySession) addFriend(command common.GameSpyCommand) {
	strNewProfileId := command.OtherValues["newprofileid"]
	newProfileId, err := strconv.ParseUint(strNewProfileId, 10, 32)
	if err != nil {
		g.replyError(1536)
		return
	}

	// Required for a friend auth
	if g.User.LastName == "" {
		logging.Error(g.ModuleName, "Add friend without last name")
		g.replyError(1537)
		return
	}

	if newProfileId == uint64(g.User.ProfileId) {
		logging.Error(g.ModuleName, "Attempt to add self as friend")
		g.replyError(1538)
		return
	}

	fc := common.CalcFriendCodeString(uint32(newProfileId), "RMCJ")
	logging.Notice(g.ModuleName, "Add friend:", aurora.Cyan(strNewProfileId), aurora.Cyan(fc))

	if g.isFriendAdded(uint32(newProfileId)) {
		g.replyError(1539)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// TODO: Add a limit
	g.FriendList = append(g.FriendList, uint32(newProfileId))
	sendMessageToProfileId("2", g.User.ProfileId, uint32(newProfileId), "\r\n\r\n|signed|"+common.RandomHexString(32))
}

func (g *GameSpySession) removeFriend(command common.GameSpyCommand) {
	// TODO
}

func (g *GameSpySession) authAddFriend(command common.GameSpyCommand) {
	strFromProfileId := command.OtherValues["fromprofileid"]
	fromProfileId, err := strconv.ParseUint(strFromProfileId, 10, 32)
	if err != nil {
		logging.Error(g.ModuleName, "Invalid profile ID string:", aurora.Cyan(strFromProfileId))
		g.replyError(1793)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	sendMessageToProfileId("4", g.User.ProfileId, uint32(fromProfileId), "")
	// Exchange statuses now
	g.exchangeFriendStatus(uint32(fromProfileId))
}

func (g *GameSpySession) setStatus(command common.GameSpyCommand) {
	status := command.CommandValue

	statstring, ok := command.OtherValues["statstring"]
	if !ok {
		logging.Notice(g.ModuleName, "Missing statstring")
		statstring = ""
	}

	locstring, ok := command.OtherValues["locstring"]
	if !ok {
		logging.Notice(g.ModuleName, "Missing locstring")
		locstring = ""
	}

	// Get the IP address for the status msg
	var rawIP int
	for i, s := range strings.Split(strings.Split(g.Conn.RemoteAddr().String(), ":")[0], ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		rawIP |= val << (24 - i*8)
	}

	ip := strconv.FormatInt(int64(int32(rawIP)), 10)

	statusMsg := "|s|" + status + "|ss|" + statstring + "|ls|" + locstring + "|ip|" + ip + "|p|0|qm|0"
	logging.Notice(g.ModuleName, "New status:", aurora.BrightMagenta(statusMsg))

	mutex.Lock()
	g.LocString = locstring
	g.Status = statusMsg

	for _, storedPid := range g.FriendList {
		g.sendFriendStatus(storedPid)
	}
	mutex.Unlock()
}

func (g *GameSpySession) bestieMessage(command common.GameSpyCommand) {
	if command.CommandValue != "1" {
		logging.Notice(g.ModuleName, "Received unknown bestie message type:", aurora.Cyan(command.CommandValue))
		return
	}

	strToProfileId := command.OtherValues["t"]
	toProfileId, err := strconv.ParseUint(strToProfileId, 10, 32)
	if err != nil {
		logging.Error(g.ModuleName, "Invalid profile ID string:", aurora.Cyan(strToProfileId))
		g.replyError(2304)
		return
	}

	if !g.isFriendAdded(uint32(toProfileId)) {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "not even on own friend list")
		g.replyError(2305)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if toSession, ok := sessions[uint32(toProfileId)]; ok && toSession.LoggedIn {
		if !toSession.isFriendAdded(g.User.ProfileId) {
			logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not friends with sender")
		}

		// TODO SECURITY: Sanitize message (there's a stack overflow exploit in DWC here)
		sendMessageToSession("1", g.User.ProfileId, toSession, command.OtherValues["msg"])
		return
	}

	logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not online")
	g.replyError(2307)
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

func sendMessageToProfileId(msgType string, from uint32, to uint32, msg string) bool {
	if session, ok := sessions[to]; ok && session.LoggedIn {
		sendMessageToSession(msgType, from, session, msg)
		return true
	}

	logging.Info("GPCM", "Destination", aurora.Cyan(to), "from", aurora.Cyan(from), "is not online")
	return false
}

func (g *GameSpySession) sendFriendStatus(profileId uint32) {
	if g.isFriendAdded(profileId) {
		if session, ok := sessions[profileId]; ok && session.LoggedIn && session.isFriendAdded(g.User.ProfileId) {
			sendMessageToSession("100", g.User.ProfileId, session, g.Status)
		}
	}
}

func (g *GameSpySession) exchangeFriendStatus(profileId uint32) {
	if g.isFriendAdded(profileId) {
		if session, ok := sessions[profileId]; ok && session.LoggedIn && session.isFriendAdded(g.User.ProfileId) {
			sendMessageToSession("100", g.User.ProfileId, session, g.Status)
			sendMessageToSession("100", profileId, g, session.Status)
		}
	}
}

func (g *GameSpySession) sendLogoutStatus() {
	// Get the IP address for the status msg
	var rawIP int
	for i, s := range strings.Split(strings.Split(g.Conn.RemoteAddr().String(), ":")[0], ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		rawIP |= val << (24 - i*8)
	}

	ip := strconv.FormatInt(int64(int32(rawIP)), 10)

	mutex.Lock()
	for _, storedPid := range g.FriendList {
		sendMessageToProfileId("100", g.User.ProfileId, storedPid, "|s|0|ss|Offline|ls||ip|"+ip+"|p|0|qm|0")
	}
	mutex.Unlock()
}

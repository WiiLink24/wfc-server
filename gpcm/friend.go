package gpcm

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"
)

func (g *GameSpySession) isFriendAdded(profileId uint32) bool {
	for _, storedPid := range g.FriendList {
		if storedPid == profileId {
			return true
		}
	}
	return false
}

func (g *GameSpySession) isFriendAuthorized(profileId uint32) bool {
	for _, storedPid := range g.AuthFriendList {
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
		g.replyError(ErrAddFriend)
		return
	}

	// Required for a friend auth
	if g.User.LastName == "" {
		logging.Error(g.ModuleName, "Add friend without last name")
		g.replyError(ErrAddFriendBadFrom)
		return
	}

	if newProfileId == uint64(g.User.ProfileId) {
		logging.Error(g.ModuleName, "Attempt to add self as friend")
		g.replyError(ErrAddFriendBadNew)
		return
	}

	fc := common.CalcFriendCodeString(uint32(newProfileId), "RMCJ")
	logging.Notice(g.ModuleName, "Add friend:", aurora.Cyan(strNewProfileId), aurora.Cyan(fc))

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
	g.FriendList = append(g.FriendList, uint32(newProfileId))

	// Check if destination has added the sender
	newSession, ok := sessions[uint32(newProfileId)]
	if !ok || newSession == nil || !newSession.LoggedIn {
		logging.Info(g.ModuleName, "Destination is not online")
		return
	}

	if !newSession.isFriendAdded(g.User.ProfileId) {
		// Not an error, just ignore for now
		logging.Info(g.ModuleName, "Destination has not added sender")
		return
	}

	// Friends are now mutual!
	// TODO: Add a limit
	g.AuthFriendList = append(g.AuthFriendList, uint32(newProfileId))

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
		g.replyError(ErrAuthAddBadFrom)
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

	statusMsg := "|s|" + status + "|ss|" + statstring + "|ls|" + locstring + "|ip|0|p|0|qm|0"
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
		g.replyError(ErrMessage)
		return
	}

	if !g.isFriendAdded(uint32(toProfileId)) {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not even on sender's friend list")
		g.replyError(ErrMessageNotFriends)
		return
	}

	msg, ok := command.OtherValues["msg"]
	if !ok || msg == "" {
		logging.Error(g.ModuleName, "Missing message value")
		g.replyError(ErrMessage)
		return
	}

	// Parse message for security and room tracking purposes
	var version int
	var msgDataIndex int

	if strings.HasPrefix(msg, "GPCM3vMAT") {
		version = 3
		msgDataIndex = 9
	} else if strings.HasPrefix(msg, "GPCM11vMAT") {
		// Only used for Brawl
		version = 11
		msgDataIndex = 10
	} else if strings.HasPrefix(msg, "GPCM90vMAT") {
		version = 90
		msgDataIndex = 10
	} else {
		logging.Error(g.ModuleName, "Invalid message prefix; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if len(msg) < msgDataIndex+1 {
		logging.Error(g.ModuleName, "Invalid message length; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	cmd := msg[msgDataIndex]
	msgDataIndex++

	var msgData []byte

	switch version {
	case 3:
		for _, stringValue := range strings.Split(msg[msgDataIndex:], "/") {
			intValue, err := strconv.ParseUint(stringValue, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Invalid message value; message:", msg)
				g.replyError(ErrMessage)
				return
			}

			msgData = binary.LittleEndian.AppendUint32(msgData, uint32(intValue))
		}
		break

	case 11:
		for _, stringValue := range strings.Split(msg[msgDataIndex:], "/") {
			byteValue, err := hex.DecodeString(stringValue)
			if err != nil || len(byteValue) != 4 {
				logging.Error(g.ModuleName, "Invalid message value; message:", msg)
				g.replyError(ErrMessage)
				return
			}

			msgData = append(msgData, byteValue...)
		}
		break

	case 90:
		msgData, err = common.Base64DwcEncoding.DecodeString(msg[msgDataIndex:])
		if err != nil {
			logging.Error(g.ModuleName, "Invalid message base64 data; message:", msg)
			g.replyError(ErrMessage)
			return
		}
		break
	}

	msgMatchData, ok := common.DecodeMatchCommand(cmd, msgData, version)
	common.LogMatchCommand(g.ModuleName, strconv.FormatInt(int64(toProfileId), 10), cmd, msgMatchData)
	if !ok {
		logging.Error(g.ModuleName, "Invalid match command data; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if cmd == common.MatchReservation {
		if common.IPFormatNoPortToInt(g.Conn.RemoteAddr().String()) == int32(msgMatchData.Reservation.PublicIP) {
			g.QR2IP = uint64(msgMatchData.Reservation.PublicIP) | (uint64(msgMatchData.Reservation.PublicPort) << 32)
		}
	} else if cmd == common.MatchResvOK {
		if common.IPFormatNoPortToInt(g.Conn.RemoteAddr().String()) == int32(msgMatchData.ResvOK.PublicIP) {
			g.QR2IP = uint64(msgMatchData.ResvOK.PublicIP) | (uint64(msgMatchData.ResvOK.PublicPort) << 32)
		}
	}

	// TODO: Replace public IP with QR2 search ID

	mutex.Lock()
	defer mutex.Unlock()

	var toSession *GameSpySession
	if toSession, ok = sessions[uint32(toProfileId)]; !ok || !toSession.LoggedIn {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not online")
		g.replyError(ErrMessageFriendOffline)
		return
	}

	if !toSession.isFriendAdded(g.User.ProfileId) {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not friends with sender")
		g.replyError(ErrMessageNotFriends)
		return
	}

	if cmd == common.MatchReservation {
		g.ReservationPID = uint32(toProfileId)
	} else if cmd == common.MatchResvOK || cmd == common.MatchResvDeny || cmd == common.MatchResvWait {
		if toSession.ReservationPID != g.User.ProfileId {
			logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "has no reservation with the sender")
			g.replyError(ErrMessage)
			return
		}

		if cmd == common.MatchResvOK {
			if g.QR2IP == 0 || toSession.QR2IP == 0 {
				logging.Error(g.ModuleName, "Missing QR2 IP")
				g.replyError(ErrMessage)
				return
			}

			if !qr2.ProcessGPResvOK(*msgMatchData.ResvOK, g.QR2IP, g.User.ProfileId, toSession.QR2IP, uint32(toProfileId)) {
				g.replyError(ErrMessage)
				return
			}
		}
	}

	sendMessageToSession("1", g.User.ProfileId, toSession, msg)
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
	mutex.Lock()
	for _, storedPid := range g.AuthFriendList {
		sendMessageToProfileId("100", g.User.ProfileId, storedPid, "|s|0|ss|Offline|ls||ip|0|p|0|qm|0")
	}
	mutex.Unlock()
}

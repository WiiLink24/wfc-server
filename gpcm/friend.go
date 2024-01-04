package gpcm

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
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

const addFriendMessage = "\r\n\r\n|signed|00000000000000000000000000000000"
const logOutMessage = "|s|0|ss|Offline|ls||ip|0|p|0|qm|0"

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
	if !g.isFriendAdded(uint32(newProfileId)) {
		g.FriendList = append(g.FriendList, uint32(newProfileId))
	}

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

	sendMessageToProfileId("2", g.User.ProfileId, uint32(newProfileId), addFriendMessage)
}

func (g *GameSpySession) sendFriendRequests() {
	mutex.Lock()
	defer mutex.Unlock()

	// Condense all requests into one packet
	var message string

	for _, newSession := range sessions {
		if newSession.isFriendAdded(g.User.ProfileId) {
			message += common.CreateGameSpyMessage(common.GameSpyCommand{
				Command:      "bm",
				CommandValue: "2",
				OtherValues: map[string]string{
					"f":   strconv.FormatUint(uint64(newSession.User.ProfileId), 10),
					"msg": addFriendMessage,
				},
			})
		}
	}

	if message != "" {
		g.Conn.Write([]byte(message))
	}
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

	if !g.isFriendAdded(delProfileID32) {
		logging.Error(g.ModuleName, aurora.Cyan(strDelProfileID), "is not a friend")
		g.replyError(ErrDeleteFriendNotFriends)
		return
	}
	if !g.isFriendAuthorized(delProfileID32) {
		logging.Error(g.ModuleName, aurora.Cyan(strDelProfileID), "is not an authorized friend")
		g.replyError(ErrDeleteFriendNotFriends)
		return
	}

	delProfileIDIndex := g.getFriendIndex(delProfileID32)
	removeFromUint32Array(&g.FriendList, delProfileIDIndex)
	delProfileIDIndex = g.getAuthorizedFriendIndex(delProfileID32)
	removeFromUint32Array(&g.AuthFriendList, delProfileIDIndex)

	sendMessageToProfileId("100", g.User.ProfileId, delProfileID32, logOutMessage)
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
	logging.Notice(g.ModuleName, "New status:", aurora.BrightMagenta(statusMsg))

	mutex.Lock()
	g.LocString = locstring
	g.Status = statusMsg

	for _, storedPid := range g.FriendList {
		g.sendFriendStatus(storedPid)
	}
	mutex.Unlock()
}

const (
	resvDenyVer3  = "GPCM3vMAT\x0316"
	resvDenyVer11 = "GPCM11vMAT\x0300000010"
	resvDenyVer90 = "GPCM90vMAT\x03EAAAAA**"

	resvWaitVer3  = "GPCM3vMAT\x04"
	resvWaitVer11 = "GPCM11vMAT\x04"
	resvWaitVer90 = "GPCM90vMAT\x04"
)

func (g *GameSpySession) bestieMessage(command common.GameSpyCommand) {
	// TODO: There are other command values that mean the same thing
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
	var resvDenyMsg string
	var resvWaitMsg string

	if strings.HasPrefix(msg, "GPCM3vMAT") {
		version = 3
		resvDenyMsg = resvDenyVer3
		resvWaitMsg = resvWaitVer3
		msgDataIndex = 9
	} else if strings.HasPrefix(msg, "GPCM11vMAT") {
		// Only used for Brawl
		version = 11
		resvDenyMsg = resvDenyVer11
		resvWaitMsg = resvWaitVer11
		msgDataIndex = 10
	} else if strings.HasPrefix(msg, "GPCM90vMAT") {
		version = 90
		resvDenyMsg = resvDenyVer90
		resvWaitMsg = resvWaitVer90
		msgDataIndex = 10
	} else {
		logging.Error(g.ModuleName, "Invalid message prefix; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if !g.DeviceAuthenticated {
		logging.Notice(g.ModuleName, "Sender is not device authenticated yet")
		// g.replyError(ErrMessage)
		sendMessageToSession("1", uint32(toProfileId), g, resvWaitMsg)
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

	case 90:
		msgData, err = common.Base64DwcEncoding.DecodeString(msg[msgDataIndex:])
		if err != nil {
			logging.Error(g.ModuleName, "Invalid message base64 data; message:", msg)
			g.replyError(ErrMessage)
			return
		}

	default:
		logging.Error(g.ModuleName, "Invalid message version; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if len(msgData) > 0x200 || (len(msgData)&3) != 0 {
		logging.Error(g.ModuleName, "Invalid length message data; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	msgMatchData, ok := common.DecodeMatchCommand(cmd, msgData, version)
	common.LogMatchCommand(g.ModuleName, strconv.FormatInt(int64(toProfileId), 10), cmd, msgMatchData)
	if !ok {
		logging.Error(g.ModuleName, "Invalid match command data; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if cmd == common.MatchReservation {
		if common.IPFormatNoPortToInt(g.Conn.RemoteAddr().String()) != int32(msgMatchData.Reservation.PublicIP) {
			logging.Error(g.ModuleName, "RESERVATION: Public IP mismatch")
			g.replyError(ErrMessage)
			return
		}

		g.QR2IP = uint64(msgMatchData.Reservation.PublicIP) | (uint64(msgMatchData.Reservation.PublicPort) << 32)

	} else if cmd == common.MatchResvOK {
		if common.IPFormatNoPortToInt(g.Conn.RemoteAddr().String()) != int32(msgMatchData.ResvOK.PublicIP) {
			logging.Error(g.ModuleName, "RESV_OK: Public IP mismatch")
			g.replyError(ErrMessage)
			return
		}

		g.QR2IP = uint64(msgMatchData.ResvOK.PublicIP) | (uint64(msgMatchData.ResvOK.PublicPort) << 32)
	}

	mutex.Lock()
	defer mutex.Unlock()

	var toSession *GameSpySession
	if toSession, ok = sessions[uint32(toProfileId)]; !ok || !toSession.LoggedIn {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not online")
		// g.replyError(ErrMessageFriendOffline)
		sendMessageToSession("1", uint32(toProfileId), g, resvDenyMsg)
		return
	}

	if !toSession.isFriendAdded(g.User.ProfileId) {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not friends with sender")
		g.replyError(ErrMessageNotFriends)
		return
	}

	if !toSession.DeviceAuthenticated {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not device authenticated")
		sendMessageToSession("1", uint32(toProfileId), g, resvDenyMsg)
		return
	}

	if cmd == common.MatchReservation {
		msgMatchData.Reservation.PublicIP = 0
		msgMatchData.Reservation.PublicPort = 0
		msgMatchData.Reservation.LocalIP = 0
		msgMatchData.Reservation.LocalPort = 0
	} else if cmd == common.MatchResvOK || cmd == common.MatchResvDeny || cmd == common.MatchResvWait {
		if toSession.ReservationPID != g.User.ProfileId || toSession.Reservation.Reservation == nil {
			logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "has no reservation with the sender")
			g.replyError(ErrMessage)
			return
		}

		if toSession.Reservation.Version != msgMatchData.Version {
			logging.Error(g.ModuleName, "Reservation version mismatch")
			g.replyError(ErrMessage)
			return
		}

		if cmd == common.MatchResvOK {
			if g.QR2IP == 0 || toSession.QR2IP == 0 {
				logging.Error(g.ModuleName, "Missing QR2 IP")
				g.replyError(ErrMessage)
				return
			}

			if !qr2.ProcessGPResvOK(msgMatchData.Version, *toSession.Reservation.Reservation, *msgMatchData.ResvOK, g.QR2IP, g.User.ProfileId, toSession.QR2IP, uint32(toProfileId)) {
				g.replyError(ErrMessage)
				return
			}

			if g.QR2IP&0xffffffff != toSession.QR2IP&0xffffffff {
				searchId := qr2.GetSearchID(g.QR2IP)
				if searchId == 0 {
					logging.Error(g.ModuleName, "Could not get QR2 search ID for IP", aurora.Cyan(fmt.Sprintf("%016x", g.QR2IP)))
					g.replyError(ErrMessage)
					return
				}

				msgMatchData.ResvOK.PublicIP = uint32(searchId & 0xffffffff)
				msgMatchData.ResvOK.PublicPort = uint16(searchId >> 32)
			}
		}
	}

	newMsg, ok := common.EncodeMatchCommand(cmd, msgMatchData)
	if !ok || len(newMsg) > 0x200 {
		logging.Error(g.ModuleName, "Failed to encode match command; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if cmd == common.MatchReservation {
		g.Reservation = msgMatchData
		g.ReservationPID = uint32(toProfileId)
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
			// Prevent players abusing a stack overflow exploit with the locstring in Mario Kart Wii
			if session.NeedsExploit && strings.HasPrefix(session.GameCode, "RMC") && len(g.LocString) > 0x14 {
				logging.Warn("GPCM", "Blocked message from", aurora.Cyan(g.User.ProfileId), "to", aurora.Cyan(session.User.ProfileId), "due to a stack overflow exploit")
				return
			}

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
		sendMessageToProfileId("100", g.User.ProfileId, storedPid, logOutMessage)
	}
	mutex.Unlock()
}

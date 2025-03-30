package gpcm

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
)

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
		logging.Error(g.ModuleName, "Received unknown bestie message type:", aurora.Cyan(command.CommandValue))
		return
	}

	strToProfileId := command.OtherValues["t"]
	toProfileId, err := strconv.ParseUint(strToProfileId, 10, 32)
	if err != nil {
		logging.Error(g.ModuleName, "Invalid profile ID string:", aurora.Cyan(strToProfileId))
		g.replyError(ErrMessage)
		return
	}

	if !g.isFriendAuthorized(uint32(toProfileId)) {
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
		sendMessageToSessionBuffer("1", uint32(toProfileId), g, resvWaitMsg)
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
		if len(msg) == msgDataIndex {
			break
		}

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
		if len(msg) == msgDataIndex {
			break
		}

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
		if len(msg) == msgDataIndex {
			break
		}

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
		g.QR2IP = uint64(msgMatchData.Reservation.PublicIP) | (uint64(msgMatchData.Reservation.PublicPort) << 32)
	} else if cmd == common.MatchResvOK {
		g.QR2IP = uint64(msgMatchData.ResvOK.PublicIP) | (uint64(msgMatchData.ResvOK.PublicPort) << 32)
	}

	mutex.Lock()
	defer mutex.Unlock()

	var toSession *GameSpySession
	if toSession, ok = sessions[uint32(toProfileId)]; !ok || !toSession.LoggedIn {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not online")
		// g.replyError(ErrMessageFriendOffline)
		sendMessageToSessionBuffer("1", uint32(toProfileId), g, resvDenyMsg)
		return
	}

	if toSession.GameName != g.GameName {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not playing the same game")
		g.replyError(ErrMessage)
		return
	}

	if !toSession.DeviceAuthenticated {
		logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "is not device authenticated")
		sendMessageToSessionBuffer("1", uint32(toProfileId), g, resvDenyMsg)
		return
	}

	sameAddress := strings.Split(g.RemoteAddr, ":")[0] == strings.Split(toSession.RemoteAddr, ":")[0]

	if cmd == common.MatchReservation {
		if g.QR2IP == 0 {
			logging.Error(g.ModuleName, "Missing QR2 IP")
			g.replyError(ErrMessage)
			return
		}

		if g.User.Restricted || toSession.User.Restricted {
			// Check with QR2 if the room is public or private
			resvError := qr2.CheckGPReservationAllowed(g.QR2IP, g.User.ProfileId, uint32(toProfileId), msgMatchData.Reservation.MatchType)
			if resvError != "ok" {
				if resvError == "restricted" || resvError == "restricted_join" {
					logging.Error(g.ModuleName, "RESERVATION: Restricted user tried to connect to public room")

					// Kick the player(s)
					if g.User.Restricted {
						kickPlayer(toSession.User.ProfileId, resvError)
					}
					if toSession.User.Restricted {
						kickPlayer(g.User.ProfileId, resvError)
					}
				}

				logging.Warn(g.ModuleName, "RESERVATION: Not allowed:", resvError)
				// Otherwise generic error?
				return
			}
		}

		if !sameAddress {
			searchId := qr2.GetSearchID(g.QR2IP)
			msgMatchData.Reservation.PublicIP = uint32(searchId & 0xffffffff)
			msgMatchData.Reservation.PublicPort = uint16(searchId >> 32)
			msgMatchData.Reservation.LocalIP = 0
			msgMatchData.Reservation.LocalPort = 0
		}
	} else if cmd == common.MatchResvOK || cmd == common.MatchResvDeny || cmd == common.MatchResvWait {
		if toSession.ReservationPID != g.User.ProfileId || toSession.Reservation.Reservation == nil {
			logging.Error(g.ModuleName, "Destination", aurora.Cyan(toProfileId), "has no reservation with the sender")
			// Allow the message through anyway to avoid a room deadlock
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

			if !sameAddress {
				searchId := qr2.GetSearchID(g.QR2IP)
				if searchId == 0 {
					logging.Error(g.ModuleName, "Could not get QR2 search ID for IP", aurora.Cyan(fmt.Sprintf("%016x", g.QR2IP)))
					g.replyError(ErrMessage)
					return
				}

				msgMatchData.ResvOK.PublicIP = uint32(searchId & 0xffffffff)
				msgMatchData.ResvOK.PublicPort = uint16(searchId >> 32)
			}
		} else if toSession.ReservationPID == g.User.ProfileId {
			toSession.ReservationPID = 0
		}
	} else if cmd == common.MatchTellAddr {
		if g.QR2IP == 0 || toSession.QR2IP == 0 {
			logging.Error(g.ModuleName, "Missing QR2 IP")
			g.replyError(ErrMessage)
			return
		}

		qr2.ProcessGPTellAddr(g.User.ProfileId, g.QR2IP, toSession.User.ProfileId, toSession.QR2IP)
	}

	newMsg, ok := common.EncodeMatchCommand(cmd, msgMatchData)
	if !ok || len(newMsg) > 0x200 || (len(newMsg)%4) != 0 {
		logging.Error(g.ModuleName, "Failed to encode match command; message:", msg)
		g.replyError(ErrMessage)
		return
	}

	if cmd == common.MatchReservation {
		g.Reservation = msgMatchData
		g.ReservationPID = uint32(toProfileId)
	}

	var newMsgStr string

	// Re-encode the new message
	switch version {
	case 3:
		newMsgStr = "GPCM3vMAT" + string(cmd)

		for i := 0; i < len(newMsg); i += 4 {
			if i > 0 {
				newMsgStr += "/"
			}

			newMsgStr += strconv.FormatUint(uint64(binary.LittleEndian.Uint32(newMsg[i:])), 10)
		}

	case 11:
		newMsgStr = "GPCM11vMAT" + string(cmd)

		for i := 0; i < len(newMsg); i += 4 {
			if i > 0 {
				newMsgStr += "/"
			}

			newMsgStr += hex.EncodeToString(newMsg[i : i+4])
		}

	case 90:
		newMsgStr = "GPCM90vMAT" + string(cmd) + common.Base64DwcEncoding.EncodeToString(newMsg)
	}

	// Check if this session is on the destination's RecvStatusFromList
	for _, friend := range toSession.RecvStatusFromList {
		if friend == g.User.ProfileId {
			// The destination has already received a status message from the sender, so we can just send the message
			sendMessageToSession("1", g.User.ProfileId, toSession, newMsgStr)
			return
		}
	}

	// Send a dummy status message so the destination will accept a message from the sender
	message := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "bm",
		CommandValue: "100",
		OtherValues: map[string]string{
			"f":   strconv.FormatUint(uint64(g.User.ProfileId), 10),
			"msg": "|s|0|ss||ls||ip|0|p|0|qm|0",
		},
	})

	message += common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "bm",
		CommandValue: "1",
		OtherValues: map[string]string{
			"f":   strconv.FormatUint(uint64(g.User.ProfileId), 10),
			"msg": newMsgStr,
		},
	})

	common.SendPacket(ServerName, toSession.ConnIndex, []byte(message))

	// Append sender's profile ID to dest's RecvStatusFromList
	toSession.RecvStatusFromList = append(toSession.RecvStatusFromList, g.User.ProfileId)

}

package gpcm

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
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

func (g *GameSpySession) addFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	strNewProfileId := command.OtherValues["newprofileid"]
	newProfileId, err := strconv.ParseUint(strNewProfileId, 10, 32)
	if err != nil {
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

func (g *GameSpySession) removeFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	// TODO
}

func (g *GameSpySession) bestieMessage(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	if command.CommandValue != "1" {
		logging.Notice(g.ModuleName, "Received unknown bestie message type:", aurora.Cyan(command.CommandValue))
		return
	}

	strToProfileId := command.OtherValues["t"]
	toProfileId, err := strconv.ParseUint(strToProfileId, 10, 32)
	if err != nil {
		g.replyError(2304)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if toSession, ok := sessions[uint32(toProfileId)]; ok && toSession.LoggedIn {
		// TODO: Check if mutual friends
		// TODO SECURITY: Sanitize message (there's a stack overflow exploit in DWC here)
		sendMessageToSession("1", g.User.ProfileId, toSession, command.OtherValues["msg"])
		return
	}

	g.replyError(2307)
}

func (g *GameSpySession) authAddFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	strFromProfileId := command.OtherValues["fromprofileid"]
	fromProfileId, err := strconv.ParseUint(strFromProfileId, 10, 32)
	if err != nil {
		g.replyError(1793)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	sendMessageToProfileId("4", g.User.ProfileId, uint32(fromProfileId), "")
	// Send status as well
	if session, ok := sessions[uint32(fromProfileId)]; ok && session.LoggedIn && g.Status != "" {
		// TODO: Check if on friend list
		session.Conn.Write([]byte(g.Status))
	}
}

func (g *GameSpySession) setStatus(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	status := command.CommandValue

	statstring, ok := command.OtherValues["statstring"]
	if !ok {
		logging.Notice(g.ModuleName, "Missing statstring")
		statstring = ""
	} else {
		if statstring == "" {
			logging.Notice(g.ModuleName, "statstring: (empty)")
		} else {
			logging.Notice(g.ModuleName, "statstring:", aurora.Cyan(statstring))
		}
	}

	locstring, ok := command.OtherValues["locstring"]
	if !ok {
		logging.Notice(g.ModuleName, "Missing locstring")
		locstring = ""
	} else {
		if locstring == "" {
			logging.Notice(g.ModuleName, "locstring: (empty)")
		} else {
			logging.Notice(g.ModuleName, "locstring:", aurora.Cyan(locstring))
		}
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

	// TODO: Check if this handles negative numbers correctly
	ip := strconv.FormatInt(int64(int32(rawIP)), 10)

	friendStatus := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "bm",
		CommandValue: "100",
		OtherValues: map[string]string{
			"f":   strconv.FormatUint(uint64(g.User.ProfileId), 10),
			"msg": "|s|" + status + "|ss|" + statstring + "|ls|" + locstring + "|ip|" + ip + "|p|0|qm|0",
		},
	})

	mutex.Lock()
	g.LocString = locstring
	g.Status = friendStatus

	for _, storedPid := range g.FriendList {
		if session, ok := sessions[uint32(storedPid)]; ok && session.LoggedIn {
			if !session.isFriendAdded(g.User.ProfileId) {
				continue
			}

			// TODO: Check if on friend list
			session.Conn.Write([]byte(friendStatus))
		}
	}

	mutex.Unlock()
}

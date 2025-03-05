package gamestats

import (
	"strconv"
	"strings"
	"time"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/jackc/pgx/v4"
	"github.com/logrusorgru/aurora/v3"
)

func (g *GameStatsSession) setpd(command common.GameSpyCommand) {
	// Example (with formatting):
	// \setpd\
	//     \pid\1000000004
	//     \ptype\3
	//     \dindex\0
	//     \kv\1
	//     \lid\0
	//     \length\149
	//     \data\
	//         \itast_friend_p\AFAAYQBsAGEAcABlAGwAaQAAAAAAANzmAFAAYQBsAGEAcABlAGwAaQAAAABSJYgbuuQEbA5pAASOoAk9JpJsjKhAFEmQTQCKAIolBAAAAAAAAAAAAAAAAAAAAAAAAAAAGps*\x00
	// \final\

	errMsg := common.GameSpyCommand{
		Command:      "setpdr",
		CommandValue: "0",
		OtherValues: map[string]string{
			"pid": command.OtherValues["pid"],
			"lid": strconv.Itoa(g.LoginID),
		},
	}

	if command.OtherValues["pid"] != strconv.FormatUint(uint64(g.User.ProfileId), 10) {
		logging.Error(g.ModuleName, "Invalid profile ID:", aurora.Cyan(command.OtherValues["pid"]))
		g.Write(errMsg)
		return
	}

	dindex, ok := command.OtherValues["dindex"]
	if !ok {
		logging.Error(g.ModuleName, "Missing dindex")
		logging.Error(g.ModuleName, "Full command:", command)
		g.Write(errMsg)
		return
	}

	ptype, ok := command.OtherValues["ptype"]
	if !ok {
		logging.Error(g.ModuleName, "Missing ptype")
		logging.Error(g.ModuleName, "Full command:", command)
		g.Write(errMsg)
		return
	}

	newData, ok := command.OtherValues["data"]
	if !ok {
		logging.Error(g.ModuleName, "Missing data")
		logging.Error(g.ModuleName, "Full command:", command)
		g.Write(errMsg)
		return
	}

	logging.Info(g.ModuleName, "Set public data: PID:", aurora.Cyan(g.User.ProfileId), "Index:", aurora.Cyan(dindex), "Type:", aurora.Cyan(ptype), "Data:", aurora.Cyan(newData))

	// Trim extra null byte at the end
	if len(newData) > 0 && newData[len(newData)-1] == 0 {
		newData = newData[:len(newData)-1]
	}

	if strings.ContainsRune(newData, 0) {
		logging.Error(g.ModuleName, "Data contains null byte")
		g.Write(errMsg)
		return
	}

	var modifiedTime time.Time
	_, _, err := database.GetGameStatsPublicData(pool, ctx, g.User.ProfileId, dindex, ptype)
	if err != nil {
		if err != pgx.ErrNoRows {
			logging.Error(g.ModuleName, "GetGameStatsPublicData returned", err)
			g.Write(errMsg)
			return
		}

		modifiedTime, err = database.CreateGameStatsPublicData(pool, ctx, g.User.ProfileId, dindex, ptype, newData)
		if err != nil {
			logging.Error(g.ModuleName, "GetGameStatsPublicData returned", err)
			g.Write(errMsg)
			return
		}
	} else {
		modifiedTime, err = database.UpdateGameStatsPublicData(pool, ctx, g.User.ProfileId, dindex, ptype, newData)
		if err != nil {
			logging.Error(g.ModuleName, "UpdateGameStatsPublicData returned", err)
			g.Write(errMsg)
			return
		}
	}

	// TODO: Is mod supposed to be the last modified time or new modified time?
	g.Write(common.GameSpyCommand{
		Command:      "setpdr",
		CommandValue: "1",
		OtherValues: map[string]string{
			"lid": strconv.Itoa(g.LoginID),
			"pid": command.OtherValues["pid"],
			"mod": strconv.Itoa(int(modifiedTime.Unix())),
		},
	})
}

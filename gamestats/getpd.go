package gamestats

import (
	"strconv"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/jackc/pgx/v4"
	"github.com/logrusorgru/aurora/v3"
)

func (g *GameStatsSession) getpd(command common.GameSpyCommand) {
	errMsg := common.GameSpyCommand{
		Command:      "getpdr",
		CommandValue: "0",
		OtherValues: map[string]string{
			"pid": command.OtherValues["pid"],
			"lid": strconv.Itoa(g.LoginID),
		},
	}

	profileIdStr, ok := command.OtherValues["pid"]
	if !ok {
		logging.Error(g.ModuleName, "Missing pid")
		logging.Error(g.ModuleName, "Full command:", command)
		g.Write(errMsg)
		return
	}

	profileId, err := strconv.ParseUint(profileIdStr, 10, 32)
	if err != nil {
		logging.Error(g.ModuleName, "Invalid pid:", aurora.Cyan(profileIdStr))
		logging.Error(g.ModuleName, "Full command:", command)
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

	logging.Info(g.ModuleName, "Get public data: PID:", aurora.Cyan(profileId), "Index:", aurora.Cyan(dindex), "Type:", aurora.Cyan(ptype))

	modifiedTime, data, err := database.GetGameStatsPublicData(pool, ctx, uint32(profileId), dindex, ptype)
	if err != nil {
		if err != pgx.ErrNoRows {
			logging.Error(g.ModuleName, "GetGameStatsPublicData returned", err)
			g.Write(errMsg)
			return
		}

		logging.Warn(g.ModuleName, "No data found")
		g.Write(errMsg)
		return
	}

	g.Write(common.GameSpyCommand{
		Command:      "getpdr",
		CommandValue: "1",
		OtherValues: map[string]string{
			"lid":    strconv.Itoa(g.LoginID),
			"pid":    command.OtherValues["pid"],
			"mod":    strconv.Itoa(int(modifiedTime.Unix())),
			"length": strconv.Itoa(len(data)),
			"data":   data,
		},
	})
}

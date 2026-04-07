package gamestats

import (
	"math/rand"
	"strconv"
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func (g *GameStatsSession) auth(command common.GameSpyCommand) {
	game := common.GetGameInfoByName(command.OtherValues["gamename"])
	if game == nil {
		g.replyError(gpcm.ErrDatabase)
		return
	}

	// TODO: Validate "response"
	g.SessionKey = rand.Int31n(290000000) + 10000000
	g.GameName = command.OtherValues["gamename"]
	g.gameInfo = game

	g.Write(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "2",
		OtherValues: map[string]string{
			"sesskey": strconv.FormatInt(int64(g.SessionKey), 10),
			"proof":   "0",
			"id":      "1",
		},
	})
}

func (g *GameStatsSession) authp(command common.GameSpyCommand) {
	lid := command.OtherValues["lid"]
	errorCmd := common.GameSpyCommand{
		Command:      "pauthr",
		CommandValue: "-3",
		OtherValues: map[string]string{
			"lid":    lid,
			"errmsg": "Invalid Validation",
		},
	}

	if lid != "" {
		var err error
		g.LoginID, err = strconv.Atoi(lid)
		if err != nil {
			logging.Error(g.ModuleName, "Error parsing login ID:", err.Error())
			g.Write(errorCmd)
			return
		}
	}

	authToken := command.OtherValues["authtoken"]
	if authToken == "" {
		logging.Error(g.ModuleName, "No authtoken provided")
		g.Write(errorCmd)
		return
	}

	authTokenObj := common.NASAuthToken{}
	err := authTokenObj.Unmarshal(authToken)
	if err != nil {
		logging.Error(g.ModuleName, "Error unmarshalling authtoken:", err.Error())
		g.Write(errorCmd)
		return
	}

	g.User, err = db.LoginUserToGameStats(authTokenObj.UserID, common.NullTerminatedString(authTokenObj.GsbrCode[:]))
	if err != nil {
		logging.Error(g.ModuleName, "Error logging in user:", err.Error())
		g.Write(errorCmd)
		return
	}

	g.ModuleName = "GSTATS:" + strconv.FormatInt(int64(g.User.ProfileId), 10)
	g.Authenticated = true

	logging.Notice(g.ModuleName, "Authenticated, game name:", aurora.Cyan(g.gameInfo.Name))

	g.Write(common.GameSpyCommand{
		Command:      "pauthr",
		CommandValue: strconv.FormatUint(uint64(g.User.ProfileId), 10),
		OtherValues: map[string]string{
			"lid": lid,
		},
	})
}

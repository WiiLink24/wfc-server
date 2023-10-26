package gpcm

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

func (g *GameSpySession) getProfile(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) string {
	strProfileId := command.OtherValues["profileid"]
	profileId, err := strconv.ParseInt(strProfileId, 10, 32)
	if err != nil {
		panic(err)
	}

	user, ok := database.GetProfile(pool, ctx, uint32(profileId))
	if !ok {
		return `\pi\\final\`
	}

	_ = common.RandomHexString(32)
	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "pi",
		CommandValue: "",
		OtherValues: map[string]string{
			"profileid":  command.OtherValues["profileid"],
			"nick":       user.UniqueNick,
			"userid":     strconv.FormatInt(user.UserId, 10),
			"email":      user.Email,
			"sig":        common.RandomHexString(32),
			"uniquenick": user.UniqueNick,
			"firstname":  user.FirstName,
			"lastname":   user.LastName,
			"pid":        "11",
			"lon":        "0.000000",
			"lat":        "0.000000",
			"loc":        "",
			"id":         command.OtherValues["id"],
		},
	})
}

func (g *GameSpySession) updateProfile(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	var firstName string
	var lastName string
	if v, ok := command.OtherValues["firstname"]; ok {
		firstName = v
	}

	if v, ok := command.OtherValues["lastname"]; ok {
		lastName = v
	}

	database.UpdateUser(pool, ctx, firstName, lastName, g.User.UserId)
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
	g.Status = friendStatus
	mutex.Unlock()
}

func (g *GameSpySession) addFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	strProfileId := command.OtherValues["newprofileid"]
	profileId, err := strconv.ParseUint(strProfileId, 10, 32)
	if err != nil {
		panic(err)
	}

	fc := common.CalcFriendCodeString(uint32(profileId), "RMCJ")
	logging.Notice(g.ModuleName, "Add friend:", aurora.Cyan(strProfileId), aurora.Cyan(fc))
	// TODO
}

func (g *GameSpySession) removeFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	// TODO
}

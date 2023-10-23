package gpcm

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

func GetProfile(session *GameSpySession, pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) string {
	strProfileId := command.OtherValues["profileid"]
	profileId, err := strconv.ParseInt(strProfileId, 10, 32)
	if err != nil {
		panic(err)
	}

	user := database.GetProfile(pool, ctx, uint32(profileId))

	_ = common.RandomHexString(32)
	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "pi",
		CommandValue: "",
		OtherValues: map[string]string{
			"profileid":  command.OtherValues["profileid"],
			"nick":       user.UniqueNick,
			"userid":     strconv.FormatInt(user.UserId, 10),
			"email":      user.Email,
			"sig":        "b126556e5ee62d4da9629dfad0f6b2a8",
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

func UpdateProfile(session *GameSpySession, pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	var firstName string
	var lastName string
	if v, ok := command.OtherValues["firstname"]; ok {
		firstName = v
	}

	if v, ok := command.OtherValues["lastname"]; ok {
		lastName = v
	}

	database.UpdateUser(pool, ctx, firstName, lastName, session.User.UserId)
}

func AddFriend(session *GameSpySession, pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	strProfileId := command.OtherValues["newprofileid"]
	profileId, err := strconv.ParseUint(strProfileId, 10, 32)
	if err != nil {
		panic(err)
	}

	fc := common.CalcFriendCodeString(uint32(profileId), "RMCJ")
	logging.Notice(session.ModuleName, "Add friend:", aurora.Cyan(strProfileId).String(), aurora.Cyan(fc).String())
	// TODO
}

func RemoveFriend(session *GameSpySession, pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	// TODO
}

func createStatus() string {
	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "bm",
		CommandValue: "100",
		OtherValues: map[string]string{
			"f":   "5",
			"msg": "|s|0|ss|Offline",
		},
	})
}

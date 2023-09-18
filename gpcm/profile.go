package gpcm

import (
	"context"
	"strconv"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

func getProfile(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) string {
	strProfileId := command.OtherValues["profileid"]
	profileId, _ := strconv.Atoi(strProfileId)

	user := database.GetProfile(pool, ctx, profileId)

	_ = common.RandomHexString(32)
	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "pi",
		CommandValue: "",
		OtherValues: map[string]string{
			"profileid":  command.OtherValues["profileid"],
			"nick":       user.UniqueNick,
			"userid":     strconv.Itoa(user.UserId),
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

func updateProfile(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	var firstName string
	var lastName string
	if v, ok := command.OtherValues["firstname"]; ok {
		firstName = v
	}

	if v, ok := command.OtherValues["lastname"]; ok {
		lastName = v
	}

	database.UpdateUser(pool, ctx, firstName, lastName, userId)
}

func addFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
	profileid := command.OtherValues["newprofileid"]

	profileid_int, err := strconv.ParseUint(profileid, 10, 32)
	if err != nil {
		logging.Notice("GPCM", "Error parsing profileid:", err.Error())
		return
	}

	fc := common.CalcFriendCodeString(uint32(profileid_int), "RMCJ")
	logging.Notice("GPCM", "Add friend:", aurora.Cyan(profileid).String(), aurora.Cyan(fc).String())
	// TODO
}

func removeFriend(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand) {
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

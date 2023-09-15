package gcsp

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"strconv"
	"wwfc/common"
	"wwfc/database"
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

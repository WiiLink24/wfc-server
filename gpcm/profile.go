package gpcm

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"wwfc/common"
)

func getProfile(pool *pgxpool.Pool, ctx context.Context, command *common.GameSpyCommand) string {
	for k, v := range command.OtherValues {
		fmt.Println(fmt.Sprintf("%s: %s", k, v))
	}

	sig := hex.EncodeToString([]byte(common.RandomString(16)))
	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "pi",
		CommandValue: "",
		OtherValues: map[string]string{
			"profileid":  command.OtherValues["profileid"],
			"nick":       "7me4ijr5sRMCJ23ul711",
			"userid":     "8467681766588",
			"email":      "7me4ijr5sRMCJ23ul711@nds",
			"sig":        sig,
			"uniquenick": "7me4ijr5sRMCJ23ul711",
			"pid":        "11",
			"lon":        "0.000000",
			"lat":        "0.000000",
			"loc":        "",
			"id":         command.OtherValues["id"],
		},
	})
}

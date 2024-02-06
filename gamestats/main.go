package gamestats

import (
	"context"
	"fmt"
	"wwfc/common"

	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool

	serverName string
	webSalt    string
	webHashPad string
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	serverName = config.ServerName
	webSalt = common.RandomString(32)
	webHashPad = common.RandomString(8)

	common.ReadGameList()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	if err != nil {
		panic(err)
	}
}

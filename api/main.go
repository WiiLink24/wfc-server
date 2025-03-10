package api

import (
	"context"
	"fmt"
	"wwfc/common"
	"wwfc/database"

	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ctx       = context.Background()
	pool      *pgxpool.Pool
	apiSecret string
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	apiSecret = config.APISecret

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

	err = database.HashInit(pool, ctx)
	if err != nil {
		panic(err)
	}
}

func Shutdown() {
}

package api

import (
	"context"
	"fmt"
	"wwfc/common"

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
}

func Shutdown() {
}

// make map string string
func mmss(data ...string) map[string]string {
	ret := make(map[string]string)

	l := len(data)

	if l%2 != 0 || l == 0 {
		panic("Length of data must be divisible by two")
	}

	for i := 0; i < l; i += 2 {
		ret[data[i]] = data[i+1]
	}

	return ret
}

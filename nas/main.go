package nas

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/nhttp"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
)

func StartServer() {
	// Get config
	config := common.GetConfig()

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

	// Finally, initialize the HTTP server
	address := config.Address + ":" + config.Port
	r := NewRoute()
	ac := r.HandleGroup("ac")
	{
		ac.HandleAction("acctcreate", acctcreate)
		ac.HandleAction("login", login)
	}

	logging.Notice("NAS", "Starting HTTP server on", address)
	log.Fatal(nhttp.ListenAndServe(address, r.Handle()))
}

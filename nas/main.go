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

	address := config.Address + ":" + config.Port
	r := NewRoute()
	ac := r.HandleGroup("ac")
	{
		ac.HandleAction("acctcreate", acctcreate)
		ac.HandleAction("login", login)
	}

	// TODO: Hack lol
	p0 := r.HandleGroup("p0")
	{
		p0.HandleAction("acctcreate", getStage1)
		p0.HandleAction("login", getStage1)
	}

	logging.Notice("NAS", "Starting HTTP server on", address)
	log.Fatal(nhttp.ListenAndServe(address, r.Handle()))
}

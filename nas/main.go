package nas

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"wwfc/common"
	"wwfc/nhttp"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
)

func checkError(err error) {
	if err != nil {
		log.Fatalf("NAS server has encountered a fatal error! Reason: %v\n", err)
	}
}

func StartServer() {
	// Get config
	config := common.GetConfig()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	checkError(err)
	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	checkError(err)

	// Finally, initialize the HTTP server
	fmt.Printf("Starting HTTP connection (%s)...\nNot using the usual port for HTTP?\nBe sure to use a proxy, otherwise the Wii can't connect!\n", "[::1]")
	r := NewRoute()
	ac := r.HandleGroup("ac")
	{
		ac.HandleAction("acctcreate", acctcreate)
		ac.HandleAction("login", login)
	}

	log.Fatal(nhttp.ListenAndServe("0.0.0.0:80", r.Handle()))
}

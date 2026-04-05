package database

import (
	"context"
	"fmt"
	"wwfc/common"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Connection struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

func Start(config common.Config) Connection {
	conn := Connection{
		ctx: context.Background(),
	}

	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	conn.pool, err = pgxpool.ConnectConfig(conn.ctx, dbConf)
	if err != nil {
		panic(err)
	}

	return conn
}

func (c *Connection) Close() {
	c.pool.Close()
}

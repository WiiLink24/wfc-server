package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	queryGsGetPublicData    = `SELECT modified_time, pdata FROM gamestats_public_data WHERE profile_id = $1 AND dindex = $2 AND ptype = $3`
	queryGsInsertPublicData = `INSERT INTO gamestats_public_data (profile_id, dindex, ptype, pdata, modified_time) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP) RETURNING modified_time`
	queryGsUpdatePublicData = `UPDATE gamestats_public_data SET pdata = $4, modified_time = CURRENT_TIMESTAMP WHERE profile_id = $1 AND dindex = $2 AND ptype = $3 RETURNING modified_time`
)

func GetGameStatsPublicData(pool *pgxpool.Pool, ctx context.Context, profileId uint32, dindex string, ptype string) (modifiedTime time.Time, publicData string, err error) {
	err = pool.QueryRow(ctx, queryGsGetPublicData, profileId, dindex, ptype).Scan(&modifiedTime, &publicData)
	return
}

func CreateGameStatsPublicData(pool *pgxpool.Pool, ctx context.Context, profileId uint32, dindex string, ptype string, publicData string) (modifiedTime time.Time, err error) {
	err = pool.QueryRow(ctx, queryGsInsertPublicData, profileId, dindex, ptype, publicData).Scan(&modifiedTime)
	return
}

func UpdateGameStatsPublicData(pool *pgxpool.Pool, ctx context.Context, profileId uint32, dindex string, ptype string, publicData string) (modifiedTime time.Time, err error) {
	err = pool.QueryRow(ctx, queryGsUpdatePublicData, profileId, dindex, ptype, publicData).Scan(&modifiedTime)
	return
}

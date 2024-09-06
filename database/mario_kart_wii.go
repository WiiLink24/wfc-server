package database

import (
	"context"
	"wwfc/common"

	"github.com/jackc/pgx/v4/pgxpool"
)

type MarioKartWiiTopTenRanking struct {
	Score      int
	PID        int
	PlayerInfo string
}

const (
	getTopTenRankingsQuery = "" +
		"SELECT score, pid, playerinfo " +
		"FROM mario_kart_wii_sake " +
		"WHERE ($1 = 0 OR regionid = $1) " +
		"AND courseid = $2 " +
		"ORDER BY score ASC " +
		"LIMIT 10"
	uploadGhostFileStatement = "" +
		"INSERT INTO mario_kart_wii_sake (regionid, courseid, score, pid, playerinfo, ghost) " +
		"VALUES ($1, $2, $3, $4, $5, $6) " +
		"ON CONFLICT (courseid, pid) DO UPDATE " +
		"SET regionid = EXCLUDED.regionid, score = EXCLUDED.score, playerinfo = EXCLUDED.playerinfo, ghost = EXCLUDED.ghost"
)

func GetMarioKartWiiTopTenRankings(pool *pgxpool.Pool, ctx context.Context, regionId common.MarioKartWiiLeaderboardRegionId,
	courseId common.MarioKartWiiCourseId) ([]MarioKartWiiTopTenRanking, error) {
	rows, err := pool.Query(ctx, getTopTenRankingsQuery, regionId, courseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topTenRankings := make([]MarioKartWiiTopTenRanking, 0, 10)
	for rows.Next() {
		var topTenRanking MarioKartWiiTopTenRanking
		err = rows.Scan(&topTenRanking.Score, &topTenRanking.PID, &topTenRanking.PlayerInfo)
		if err != nil {
			return nil, err
		}

		topTenRankings = append(topTenRankings, topTenRanking)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return topTenRankings, nil
}

func UploadMarioKartWiiGhostFile(pool *pgxpool.Pool, ctx context.Context, regionId common.MarioKartWiiLeaderboardRegionId,
	courseId common.MarioKartWiiCourseId, score int, pid int, playerInfo string, ghost []byte) error {
	_, err := pool.Exec(ctx, uploadGhostFileStatement, regionId, courseId, score, pid, playerInfo, ghost)

	return err
}

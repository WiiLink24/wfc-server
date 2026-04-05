package database

import (
	"wwfc/common"

	"github.com/jackc/pgx/v4"
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
	getGhostDataQuery = "" +
		"SELECT id " +
		"FROM mario_kart_wii_sake " +
		"WHERE courseid = $1 " +
		"AND score < $2 " +
		"ORDER BY score DESC " +
		"LIMIT 1"
	getStoredGhostDataQuery = "" +
		"SELECT pid, id " +
		"FROM mario_kart_wii_sake " +
		"WHERE ($1 = 0 OR regionid = $1) " +
		"AND courseid = $2 " +
		"ORDER BY score ASC " +
		"LIMIT 1"
	getFileQuery = "" +
		"SELECT ghost " +
		"FROM mario_kart_wii_sake " +
		"WHERE id = $1 " +
		"LIMIT 1"
	getGhostFileQuery = "" +
		"SELECT ghost " +
		"FROM mario_kart_wii_sake " +
		"WHERE courseid = $1 " +
		"AND score < $2 " +
		"AND pid <> $3 " +
		"ORDER BY score DESC " +
		"LIMIT 1"
	insertGhostFileStatement = "" +
		"INSERT INTO mario_kart_wii_sake (regionid, courseid, score, pid, playerinfo, ghost, upload_time) " +
		"VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP) " +
		"ON CONFLICT (courseid, pid) DO UPDATE " +
		"SET regionid = EXCLUDED.regionid, score = EXCLUDED.score, playerinfo = EXCLUDED.playerinfo, ghost = EXCLUDED.ghost, upload_time = CURRENT_TIMESTAMP"
)

func (c *Connection) GetMarioKartWiiTopTenRankings(regionId common.MarioKartWiiLeaderboardRegionId,
	courseId common.MarioKartWiiCourseId) ([]MarioKartWiiTopTenRanking, error) {
	rows, err := c.pool.Query(c.ctx, getTopTenRankingsQuery, regionId, courseId)
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
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return topTenRankings, nil
}

func (c *Connection) GetMarioKartWiiGhostData(courseId common.MarioKartWiiCourseId, time int) (int, error) {
	row := c.pool.QueryRow(c.ctx, getGhostDataQuery, courseId, time)

	var fileId int
	if err := row.Scan(&fileId); err != nil {
		return 0, err
	}

	return fileId, nil
}

func (c *Connection) GetMarioKartWiiStoredGhostData(regionId common.MarioKartWiiLeaderboardRegionId,
	courseId common.MarioKartWiiCourseId) (int, int, error) {
	row := c.pool.QueryRow(c.ctx, getStoredGhostDataQuery, regionId, courseId)

	var pid int
	var fileId int
	if err := row.Scan(&pid, &fileId); err != nil {
		return 0, 0, err
	}

	return pid, fileId, nil
}

func (c *Connection) GetMarioKartWiiFile(fileId int) ([]byte, error) {
	row := c.pool.QueryRow(c.ctx, getFileQuery, fileId)

	var file []byte
	if err := row.Scan(&file); err != nil {
		return nil, err
	}

	return file, nil
}

func (c *Connection) GetMarioKartWiiGhostFile(courseId common.MarioKartWiiCourseId, time int, pid int) ([]byte, error) {
	row := c.pool.QueryRow(c.ctx, getGhostFileQuery, courseId, time, pid)

	var ghost []byte
	if err := row.Scan(&ghost); err != nil {
		return nil, err
	}

	return ghost, nil
}

func (c *Connection) InsertMarioKartWiiGhostFile(regionId common.MarioKartWiiLeaderboardRegionId,
	courseId common.MarioKartWiiCourseId, score int, pid int, playerInfo string, ghost []byte) error {
	_, err := c.pool.Exec(c.ctx, insertGhostFileStatement, regionId, courseId, score, pid, playerInfo, ghost)

	return err
}

// Mario Kart Wii friend info functions for API compatibility

func (c *Connection) GetMKWFriendInfo(profileId uint32) string {
	records, err := c.GetSakeRecords(1687, []int32{int32(profileId)}, "FriendInfo", nil, []string{"info"}, "")
	if err != nil || len(records) == 0 {
		return ""
	}

	infoField, ok := records[0].Fields["info"]
	if !ok {
		return ""
	}

	return infoField.Value
}

func (c *Connection) UpdateMKWFriendInfo(profileId uint32, info string) {
	records, err := c.GetSakeRecords(1687, []int32{int32(profileId)}, "FriendInfo", nil, []string{"info"}, "")
	if err == pgx.ErrNoRows || (err == nil && len(records) == 0) {
		// No existing record, insert new one
		record := SakeRecord{
			GameId:  1687,
			TableId: "FriendInfo",
			OwnerId: int32(profileId),
			Fields: map[string]SakeField{
				"info": {
					Type:  SakeFieldTypeBinaryData,
					Value: info,
				},
			},
		}
		_, err = c.InsertSakeRecord(record)
	} else if err == nil {
		// Update existing record
		records[0].Fields["info"] = SakeField{
			Type:  SakeFieldTypeBinaryData,
			Value: info,
		}
		err = c.UpdateSakeRecord(records[0], int32(profileId))

	}

	if err != nil {
		panic(err)
	}
}

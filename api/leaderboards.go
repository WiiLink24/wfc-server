package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"wwfc/logging"
)

type Leaderboard struct {
	PlayerName string `json:"player"`
	CourseId   int    `json:"course"`
	Score      int    `json:"score"`
}

func HandleMKWiiLeaderboards(w http.ResponseWriter, r *http.Request) {

	// Ejemplo: consultar todas las tuplas de la tabla "users"
	rows, err := pool.Query(ctx, "SELECT u.last_ingamesn, t.courseId, t.score FROM users u JOIN mario_kart_wii_sake ms1 ON (u.profile_id = ms1.pid) JOIN (SELECT courseId, MIN(score) AS score FROM mario_kart_wii_sake ms2 GROUP BY courseId) t ON (t.score = ms1.score AND t.courseId = ms1.courseId) ORDER BY courseId")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-LB", "Error querying users: %v", err)
		return
	}

	var leaderboards []Leaderboard
	for rows.Next() {
		var playerName string
		var courseId int
		var score int

		if err := rows.Scan(&playerName, &courseId, &score); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-LB", "Error scanning users: %v", err)
			return
		}

		// Convert score from milliseconds to MINUTOS:SEGUNDOS.MILISEGUNDOS
		//duration := time.Duration(score) * time.Millisecond
		//minutes := int(duration.Minutes())
		//seconds := int(duration.Seconds()) % 60
		//milliseconds := int(duration.Milliseconds()) % 1000
		//formattedScore := fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)

		leaderboard := Leaderboard{
			PlayerName: playerName,
			CourseId:   courseId,
			Score:      score,
		}

		leaderboards = append(leaderboards, leaderboard)
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-LB", "Error iterating users: %v", err)
		return
	}

	jsonData, err := json.Marshal(leaderboards)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-LB", "Error marshalling users: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}

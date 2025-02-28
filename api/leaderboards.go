package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"wwfc/common"
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

type Registered struct {
	Players int `json:"players"`
	Ghosts  int `json:"ghosts"`
}

func HandleRegisteredAccounts(w http.ResponseWriter, r *http.Request) {

	// Query to get the number of players
	playerRows, err := pool.Query(ctx, "SELECT COUNT(*) FROM users")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-RA", "Error querying users: %v", err)
		return
	}

	var playerCount int
	for playerRows.Next() {
		if err := playerRows.Scan(&playerCount); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-RA", "Error scanning users: %v", err)
			return
		}
	}

	defer playerRows.Close()

	if err := playerRows.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-RA", "Error iterating users: %v", err)
		return
	}

	// Query to get the number of ghosts
	ghostRows, err := pool.Query(ctx, "SELECT COUNT(*) FROM mario_kart_wii_sake")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-RA", "Error querying ghosts: %v", err)
		return
	}

	var ghostCount int
	for ghostRows.Next() {
		if err := ghostRows.Scan(&ghostCount); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-RA", "Error scanning ghosts: %v", err)
			return
		}
	}

	defer ghostRows.Close()

	if err := ghostRows.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-RA", "Error iterating ghosts: %v", err)
		return
	}

	registered := Registered{
		Players: playerCount,
		Ghosts:  ghostCount,
	}

	jsonData, err := json.Marshal(registered)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-RA", "Error marshalling registered data: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}

func HandleGetMii(w http.ResponseWriter, r *http.Request) {
	playername := r.URL.Query().Get("playername")
	courseId := r.URL.Query().Get("courseId")
	score := r.URL.Query().Get("score")

	// check query
	if playername == "" || len(playername) > 30 {
		w.WriteHeader(http.StatusBadRequest)
		logging.Info("WEB-GM", "Playername is empty or too long")
		return
	}
	courseInt, err := strconv.Atoi(courseId)
	if err != nil || courseInt < 0 || courseInt > 31 {
		w.WriteHeader(http.StatusBadRequest)
		logging.Info("WEB-GM", "Invalid course value")
		return
	}
	scoreInt, err := strconv.Atoi(score)
	if err != nil || scoreInt <= 0 || scoreInt >= 360000 {
		w.WriteHeader(http.StatusBadRequest)
		logging.Info("WEB-GM", "Invalid score value")
		return
	}

	// extract data from SQL DB
	playerRows, err := pool.Query(ctx, "SELECT ghost FROM mario_kart_wii_sake WHERE courseid = $1 AND score = $2", courseInt, scoreInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-GM", "Error querying ghosts: %v", err)
		return
	}
	var ghost []byte
	for playerRows.Next() {
		if err := playerRows.Scan(&ghost); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error scanning ghosts: %v", err)
			return
		}
	}

	// check if empty response
	if len(ghost) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-GM", "Ghost data is empty")
		return
	}

	// Convert ghost to RKGhostData.
	rkg := common.RKGhostData(ghost)

	//check if mii folder exists
	miiFolder := "./mii"
	if _, err := os.Stat(miiFolder); os.IsNotExist(err) {
		if err := os.Mkdir(miiFolder, os.ModePerm); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error creating Mii folder:", err)
			return
		}
	}

	// check if playername.png exists in the mii cache folder
	miiFilePath := miiFolder + "/" + playername + ".png"
	if _, err := os.Stat(miiFilePath); os.IsNotExist(err) {

		// Not in cache...

		logging.Info("WEB-GM", "Mii image not found in cache, accessing Mii API")

		// Extract Mii data
		miiData := rkg.GetMiiData()

		// Convert Mii data to binary
		miiBinary := miiData[:]
		if len(miiBinary) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Extracted Mii data is empty")
			return
		}

		// Convert Mii data to Base64
		miiBase64 := base64.StdEncoding.EncodeToString(miiBinary)
		if miiBase64 == "" {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error converting Mii data to Base64")
			return
		}

		// Do a POST to API mii-unsecure.ariankordi.net/miis/image.png with parameter data as the Base64 string
		// and get the image as a response
		url := "https://mii-unsecure.ariankordi.net/miis/image.png"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(miiBase64)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error creating request: %v", err)
			return
		}

		// Set headers
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		q := req.URL.Query()
		q.Add("data", miiBase64)
		q.Add("type", "face")
		q.Add("expression", "smile")
		q.Add("width", "256")
		q.Add("mipmapEnable", "true")
		q.Add("resourceType", "very_high")
		q.Add("shaderType", "wiiu_blinn")
		req.URL.RawQuery = q.Encode()

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error making request: %v", err)
			return
		}
		defer resp.Body.Close()

		// handle errors and save image
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Mii API returned non-200 status: %v", resp.StatusCode)
			return
		}
		imageData, err := io.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error reading response body: %v", err)
			return
		}
		if len(imageData) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Mii API returned empty image data")
			return
		}
		if err := os.WriteFile(miiFilePath, imageData, os.ModePerm); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error saving Mii image: %v", err)
			return
		}

		// Mii now in cache
	} else {
		// In cache...
		logging.Info("WEB-GM", "Loading Mii image from cache")
	}

	// Read image from cache and return it
	imageData, err := os.ReadFile(miiFilePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logging.Error("WEB-GM", "Error reading Mii image: %v", err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Write(imageData)
}

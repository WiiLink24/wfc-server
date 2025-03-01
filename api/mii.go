package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"wwfc/common"
	"wwfc/logging"
)

func HandleGetMii(w http.ResponseWriter, r *http.Request) {
	// Retrieve player name
	playername := r.URL.Query().Get("playername")
	// Retrieve course ID
	courseId := r.URL.Query().Get("courseId")
	courseInt, err := strconv.Atoi(courseId)
	// Retrieve score
	score := r.URL.Query().Get("score")
	scoreInt, err2 := strconv.Atoi(score)

	//check if mii folder exists
	if _, err := os.Stat("./mii"); os.IsNotExist(err) {
		if err := os.Mkdir("./mii", os.ModePerm); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logging.Error("WEB-GM", "Error creating Mii folder:", err)
			return
		}
	}

	// check if playername is set, or courseId + Score is set
	if playername != "" {
		// --- Asking for playername mode ---
		// check if playername is too long
		if len(playername) > 30 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Playername is empty or too long"))
			logging.Info("WEB-GM", "Playername is empty or too long")
			return
		}

		miiImage := getMiiCached(playername)
		if miiImage == nil {
			// Not in cache...
			// Search for any ghost from the player
			courseInt, scoreInt, err = searchAnyPlayerGhost(playername)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Player has not sent any ghosts. Unable to retrieve Mii."))
				logging.Error("WEB-GM", "Error searching any player ghost:", err)
				return
			}

			// Get Mii Base64 from ghost data
			miiBase64, err := getMiiFromGhost(courseInt, scoreInt)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error getting Mii from ghost data"))
				logging.Error("WEB-GM", "Error getting Mii from ghost data:", err)
				return
			}

			// Render Mii image
			logging.Info("WEB-GM", "POSTing Mii Render service for player", playername, "(playername mode)")
			miiImage, err = renderMii(miiBase64)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error rendering Mii image"))
				logging.Error("WEB-GM", "Error rendering Mii image:", err)
				return
			}

			// Save the Mii image to cache
			if err := os.WriteFile("./mii/"+playername+".png", miiImage, os.ModePerm); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logging.Error("WEB-GM", "Error saving Mii image:", err)
				return
			}
		}

		// Return the Mii image
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(miiImage)))
		w.Write(miiImage)
		return

	} else {
		// --- Asking for course+time mode ---
		if err != nil || courseInt < 0 || courseInt > 31 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid course value"))
			logging.Info("WEB-GM", "Invalid course value")
			return
		}
		if err2 != nil || scoreInt <= 0 || scoreInt >= 360000 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid score value"))
			logging.Info("WEB-GM", "Invalid score value")
			return
		}

		// Get Mii Base64 from ghost data
		miiBase64, err := getMiiFromGhost(courseInt, scoreInt)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error getting Mii from ghost data"))
			logging.Error("WEB-GM", "Error getting Mii from ghost data:", err)
			return
		}

		// Get playername from Mii
		unsafePlayername, err := getPlayerNameFromMii(miiBase64)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error getting playername from Mii"))
			logging.Error("WEB-GM", "Error getting playername from Mii")
			return
		}

		// Sanitize playername for file name
		playername := sanitizeString(unsafePlayername)

		miiImage := getMiiCached(playername)
		if miiImage == nil {
			// Not in cache...
			// Render Mii image
			logging.Info("WEB-GM", "POSTing Mii Render service for player", playername, "(course+score mode)")
			miiImage, err = renderMii(miiBase64)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error rendering Mii image"))
				logging.Error("WEB-GM", "Error rendering Mii image:", err)
				return
			}

			// Save the Mii image to cache
			if err := os.WriteFile("./mii/"+playername+".png", miiImage, os.ModePerm); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logging.Error("WEB-GM", "Error saving Mii image:", err)
				return
			}
		}

		// Return the Mii image
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(miiImage)))
		w.Write(miiImage)
		return

	}
}

func sanitizeString(name string) string {
	// Convert from UTF16 to UTF8
	utf16Name := []rune(name)
	utf8Name := string(utf16Name)

	// Replace any non-alphanumeric character with an empty string
	re := regexp.MustCompile(`[^\w]`)
	return re.ReplaceAllString(utf8Name, "")
}

func getMiiCached(playername string) []byte {
	// check if playername.png exists in the mii cache folder
	miiFilePath := "./mii/" + playername + ".png"
	if _, err := os.Stat(miiFilePath); os.IsNotExist(err) {
		logging.Info("WEB-GM", "Not in cache:", err)
		// Not in cache...
		return nil
	} else {
		// In cache... return the raw image
		imageData, _ := os.ReadFile(miiFilePath)
		return imageData
	}
}

func renderMii(base string) ([]byte, error) {
	// Do a POST to API mii-unsecure.ariankordi.net/miis/image.png with parameter data as the Base64 string
	// and get the image as a response
	url := "https://mii-unsecure.ariankordi.net/miis/image.png"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(base)))
	if err != nil {
		return nil, errors.New("Error creating request (" + err.Error() + ")")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	q := req.URL.Query()
	q.Add("data", base)
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
		return nil, errors.New("Error creating request (" + err.Error() + ")")
	}
	defer resp.Body.Close()

	// handle errors and save image
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Error creating request (" + strconv.Itoa(resp.StatusCode) + ")")
	}
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("Error reading response body (" + err.Error() + ")")
	}
	if len(imageData) == 0 {
		return nil, errors.New("mii API returned empty image data")
	}

	return imageData, nil
}

func searchAnyPlayerGhost(playername string) (int, int, error) {
	// extract data from SQL DB
	playerRows, err := pool.Query(ctx, "SELECT courseid, score FROM mario_kart_wii_sake WHERE pid = (SELECT profile_id FROM users WHERE last_ingamesn = $1) LIMIT 1", playername)
	if err != nil {
		return 0, 0, errors.New("Error querying ghosts (" + err.Error() + ")")
	}
	var courseid int
	var score int
	for playerRows.Next() {
		if err := playerRows.Scan(&courseid, &score); err != nil {
			logging.Error("WEB-GM", "Error scanning ghosts:", err)
			return 0, 0, errors.New("Error scanning ghosts (" + err.Error() + ")")
		}
	}
	// check if empty response
	if courseid == 0 || score == 0 {
		return 0, 0, errors.New("no results from database")
	}

	return courseid, score, nil
}

func getMiiFromGhost(course int, score int) (string, error) {
	// extract data from SQL DB
	playerRows, err := pool.Query(ctx, "SELECT ghost FROM mario_kart_wii_sake WHERE courseid = $1 AND score = $2", course, score)
	if err != nil {
		return "", errors.New("Error querying ghosts (" + err.Error() + ")")
	}
	var ghost []byte
	for playerRows.Next() {
		if err := playerRows.Scan(&ghost); err != nil {
			logging.Error("WEB-GM", "Error scanning ghosts:", err)
			return "", errors.New("Error scanning ghosts (" + err.Error() + ")")
		}
	}
	// check if empty response
	if len(ghost) == 0 {
		return "", errors.New("ghost data is empty")
	}

	// Convert ghost to RKGhostData.
	rkg := common.RKGhostData(ghost)

	// Extract Mii data
	miiData := rkg.GetMiiData()

	// Convert Mii data to binary
	miiBinary := miiData[:]
	if len(miiBinary) == 0 {
		return "", errors.New("extracted Mii data is empty")
	}

	// Convert Mii binary to Base64
	miiBase64 := base64.StdEncoding.EncodeToString(miiBinary)
	if miiBase64 == "" {
		return "", errors.New("error converting Mii data to Base64")
	}

	return miiBase64, nil
}

func getPlayerNameFromMii(base string) (string, error) {
	// Convert Base64 to Mii binary
	miiBinary, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		return "", errors.New("error decoding Mii Base64")
	}

	// Convert Mii binary to Mii data
	miiData := common.Mii(miiBinary)

	// Get playername from Mii data
	return miiData.GetPlayerName(), nil
}

/* func HandleGetMii(w http.ResponseWriter, r *http.Request) {
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
} */

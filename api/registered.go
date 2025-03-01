package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"wwfc/logging"
)

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

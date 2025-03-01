package api

import (
	"net/http"
	"strconv"
	"wwfc/logging"
)

func HandleGetMiiRaw(w http.ResponseWriter, r *http.Request) {
	// Retrieve player name
	playername := r.URL.Query().Get("playername")
	// check if secret API key is present
	if apiSecret == "" || r.URL.Query().Get("secret") != apiSecret {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid API secret"))
		logging.Info("WEB-GM", "Invalid API secret")
		return
	}

	// check if playername is set
	if playername != "" {
		// check if playername is too long
		if len(playername) > 30 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Playername is empty or too long"))
			logging.Info("WEB-GM", "Playername is empty or too long")
			return
		}

		courseInt, scoreInt, err := searchAnyPlayerGhost(playername)
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

		// Return the Mii Base64 string
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(miiBase64)))
		w.Write([]byte(miiBase64))
		return
	} else {
		// plaeryname is not set
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Playername is empty or too long"))
		logging.Info("WEB-GM", "Playername is empty or too long")
		return
	}
}

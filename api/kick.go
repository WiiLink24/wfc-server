package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"wwfc/gpcm"
)

func HandleKick(w http.ResponseWriter, r *http.Request) {
	errorString := handleKickImpl(w, r)
	if errorString != "" {
		jsonData, _ := json.Marshal(map[string]string{"error": errorString})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
		w.Write(jsonData)
	} else {
		jsonData, _ := json.Marshal(map[string]string{"success": "true"})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
		w.Write(jsonData)
	}
}

func handleKickImpl(w http.ResponseWriter, r *http.Request) string {
	// TODO: Actual authentication rather than a fixed secret
	// TODO: Use POST instead of GET

	u, err := url.Parse(r.URL.String())
	if err != nil {
		return "Bad request"
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "Bad request"
	}

	if apiSecret == "" || query.Get("secret") != apiSecret {
		return "Invalid API secret"
	}

	pidStr := query.Get("pid")
	if pidStr == "" {
		return "Missing pid in request"
	}

	pid, err := strconv.ParseUint(pidStr, 10, 32)
	if err != nil {
		return "Invalid pid"
	}

	gpcm.KickPlayer(uint32(pid), "moderator_kick")
	return ""
}

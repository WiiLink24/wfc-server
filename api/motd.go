package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/gpcm"
)

func HandleMotd(w http.ResponseWriter, r *http.Request) {
	var motd string
	var success bool
	var err string
	var statusCode int

	switch r.Method {
	case http.MethodHead:
		statusCode = http.StatusOK
	case http.MethodGet:
		_motd, motdErr := gpcm.GetMessageOfTheDay()
		if motdErr != nil {
			err = motdErr.Error()
			statusCode = http.StatusInternalServerError
			break
		}

		motd = _motd
		success = true
		statusCode = http.StatusOK
	case http.MethodPost:
		success, err, statusCode = handleMotdImpl(w, r)
	default:
		err = "Incorrect request. POST, GET, or HEAD only."
		statusCode = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodHead {
		w.WriteHeader(statusCode)
	} else {
		json, _ := json.Marshal(MotdResponse{motd, success, err})
		w.Header().Set("Content-Length", strconv.Itoa(len(json)))
		w.WriteHeader(statusCode)
		w.Write(json)
	}
}

type MotdRequestSpec struct {
	Secret string
	Motd   string
}

type MotdResponse struct {
	Motd    string
	Success bool
	Error   string
}

func handleMotdImpl(w http.ResponseWriter, r *http.Request) (bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, "Unable to read request body", http.StatusBadRequest
	}

	var req MotdRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return false, "Invalid API secret in request", http.StatusUnauthorized
	}

	err = gpcm.SetMessageOfTheDay(req.Motd)
	if err != nil {
		return false, err.Error(), http.StatusInternalServerError
	}

	// Don't return empty JSON, this is placeholder for now.
	return true, "", http.StatusOK
}

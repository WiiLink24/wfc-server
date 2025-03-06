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

	if r.Method == http.MethodPost {
		success, err, statusCode = handleMotdImpl(r)
	} else if r.Method == http.MethodGet {
		_motd, motdErr := gpcm.GetMessageOfTheDay()
		if motdErr != nil {
			err = motdErr.Error()
			statusCode = http.StatusInternalServerError
		} else {
			motd = _motd
			success = true
			statusCode = http.StatusOK
		}
	} else if r.Method == http.MethodOptions {
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	} else {
		err = "Incorrect request. POST and GET only."
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST, GET")
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(MotdResponse{motd, success, err})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
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

func handleMotdImpl(r *http.Request) (bool, string, int) {
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

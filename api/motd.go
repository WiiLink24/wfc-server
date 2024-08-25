package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/gpcm"
)

func HandleMotd(w http.ResponseWriter, r *http.Request) {
	var jsonData map[string]string
	var statusCode int

	switch r.Method {
	case http.MethodHead:
		statusCode = http.StatusOK
	case http.MethodGet:
		motd, err := gpcm.GetMessageOfTheDay()
		if err != nil {
			jsonData = mmss("error", err.Error())
			statusCode = http.StatusInternalServerError
			break
		}

		jsonData = mmss("motd", motd)
		statusCode = http.StatusOK
	case http.MethodPost:
		jsonData, statusCode = handleMotdImpl(w, r)
	default:
		jsonData = mmss("error", "Incorrect request. POST, GET, or HEAD only.")
		statusCode = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if len(jsonData) == 0 {
		w.WriteHeader(statusCode)
	} else {
		json, _ := json.Marshal(jsonData)
		w.Header().Set("Content-Length", strconv.Itoa(len(json)))
		w.WriteHeader(statusCode)
		w.Write(json)
	}
}

type MotdRequestSpec struct {
	Secret string
	Motd   string
}

func handleMotdImpl(w http.ResponseWriter, r *http.Request) (map[string]string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return mmss("error", "Unable to read request body"), http.StatusBadRequest
	}

	var req MotdRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return mmss("error", err.Error()), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return mmss("error", "Invalid API secret in request"), http.StatusUnauthorized
	}

	err = gpcm.SetMessageOfTheDay(req.Motd)
	if err != nil {
		return mmss("error", err.Error()), http.StatusInternalServerError
	}

	// Don't return empty JSON, this is placeholder for now.
	return mmss("result", "success"), http.StatusOK
}

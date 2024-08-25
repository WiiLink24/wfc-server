package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

func HandleUnban(w http.ResponseWriter, r *http.Request) {
	var jsonData map[string]string
	var statusCode int

	switch r.Method {
	case http.MethodHead:
		statusCode = http.StatusOK
	case http.MethodPost:
		jsonData, statusCode = handleUnbanImpl(w, r)
	default:
		jsonData = mmss("error", "Incorrect request. POST or HEAD only.")
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

type UnbanRequestSpec struct {
	Secret string
	Pid    uint32
}

func handleUnbanImpl(w http.ResponseWriter, r *http.Request) (map[string]string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return mmss("error", "Unable to read request body"), http.StatusBadRequest
	}

	var req UnbanRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return mmss("error", err.Error()), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return mmss("error", "Invalid API secret in request"), http.StatusUnauthorized
	}

	if req.Pid == 0 {
		return mmss("error", "pid missing or 0 in request"), http.StatusBadRequest
	}

	if !database.UnbanUser(pool, ctx, req.Pid) {
		return mmss("error", "Failed to unban user"), http.StatusInternalServerError
	}

	ip := database.GetUserIP(pool, ctx, req.Pid)
	return mmss("result", "success", "ip", ip), http.StatusOK
}

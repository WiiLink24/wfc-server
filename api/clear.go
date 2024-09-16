package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

func HandleClear(w http.ResponseWriter, r *http.Request) {
	var user *database.User
	var success bool
	var err string
	var statusCode int

	switch r.Method {
	case http.MethodHead:
		statusCode = http.StatusOK
	case http.MethodPost:
		user, success, err, statusCode = handleClearImpl(w, r)
	default:
		err = "Incorrect request. POST or HEAD only."
		statusCode = http.StatusBadRequest
	}

	if user == nil {
		user = &database.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodHead {
		w.WriteHeader(statusCode)
	} else {
		json, _ := json.Marshal(UserActionResponse{*user, success, err})
		w.Header().Set("Content-Length", strconv.Itoa(len(json)))
		w.WriteHeader(statusCode)
		w.Write(json)
	}
}

type ClearRequestSpec struct {
	Secret string
	Pid    uint32
}

func handleClearImpl(w http.ResponseWriter, r *http.Request) (*database.User, bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false, "Unable to read request body", http.StatusBadRequest
	}

	var req ClearRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return nil, false, "Invalid API secret in request", http.StatusUnauthorized
	}

	user, success := database.ClearProfile(pool, ctx, req.Pid)

	if !success {
		return nil, false, "Unable to query user data from the database", http.StatusInternalServerError
	}

	// Don't return empty JSON, this is placeholder for now.
	return &user, true, "", http.StatusOK
}

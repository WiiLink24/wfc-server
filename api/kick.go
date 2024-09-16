package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
	"wwfc/gpcm"
)

func HandleKick(w http.ResponseWriter, r *http.Request) {
	var user *database.User
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		user, success, err, statusCode = handleKickImpl(w, r)
	} else {
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

type KickRequestSpec struct {
	Secret string
	Pid    uint32
}

func handleKickImpl(w http.ResponseWriter, r *http.Request) (*database.User, bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false, "Unable to read request body", http.StatusBadRequest
	}

	var req KickRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return nil, false, "Invalid API secret in request", http.StatusUnauthorized
	}

	if req.Pid == 0 {
		return nil, false, "pid missing or 0 in request", http.StatusBadRequest
	}

	gpcm.KickPlayer(req.Pid, "moderator_kick")

	var message string
	user, success := database.GetProfile(pool, ctx, req.Pid)

	if success {
		message = ""
	} else {
		message = "Unable to query user data from the database"
	}

	return &user, success, message, http.StatusOK
}

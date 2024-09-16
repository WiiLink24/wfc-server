package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
	"wwfc/database"
	"wwfc/gpcm"
)

func HandleBan(w http.ResponseWriter, r *http.Request) {
	var user *database.User
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		user, success, err, statusCode = handleBanImpl(w, r)
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

type BanRequestSpec struct {
	Secret       string
	Pid          uint32
	Days         uint64
	Hours        uint64
	Minutes      uint64
	Tos          bool
	Reason       string
	ReasonHidden string
	Moderator    string
}

func handleBanImpl(w http.ResponseWriter, r *http.Request) (*database.User, bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false, "Unable to read request body", http.StatusBadRequest
	}

	var req BanRequestSpec
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

	if req.Reason == "" {
		return nil, false, "Missing ban reason in request", http.StatusBadRequest
	}

	moderator := req.Moderator
	if moderator == "" {
		moderator = "admin"
	}

	minutes := req.Days*24*60 + req.Hours*60 + req.Minutes
	if minutes == 0 {
		return nil, false, "Ban length missing or 0", http.StatusBadRequest
	}

	length := time.Duration(minutes) * time.Minute

	if !database.BanUser(pool, ctx, req.Pid, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		return nil, false, "Failed to ban user", http.StatusInternalServerError
	}

	gpcm.KickPlayerCustomMessage(req.Pid, req.Reason, gpcm.WWFCErrorMessage{
		ErrorCode: 22002,
		MessageRMC: map[byte]string{
			gpcm.LangEnglish: "" +
				"You have been banned from Retro WFC\n" +
				"Reason: " + req.Reason + "\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
		},
	})

	var message string
	user, success := database.GetProfile(pool, ctx, req.Pid)

	if success {
		message = ""
	} else {
		message = "Unable to query user data from the database"
	}

	return &user, success, message, http.StatusOK
}

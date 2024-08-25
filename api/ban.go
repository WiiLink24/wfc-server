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
	var jsonData map[string]string
	var statusCode int

	switch r.Method {
	case http.MethodHead:
		statusCode = http.StatusOK
	case http.MethodPost:
		jsonData, statusCode = handleBanImpl(w, r)
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

func handleBanImpl(w http.ResponseWriter, r *http.Request) (map[string]string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return mmss("error", "Unable to read request body"), http.StatusBadRequest
	}

	var req BanRequestSpec
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

	if req.Reason == "" {
		return mmss("error", "Missing ban reason in request"), http.StatusBadRequest
	}

	moderator := req.Moderator
	if moderator == "" {
		moderator = "admin"
	}

	minutes := req.Days*24*60 + req.Hours*60 + req.Minutes
	if minutes == 0 {
		return mmss("error", "Ban length missing or 0"), http.StatusBadRequest
	}

	length := time.Duration(minutes) * time.Minute

	if !database.BanUser(pool, ctx, req.Pid, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		return mmss("error", "Failed to ban user"), http.StatusInternalServerError
	}

	if req.Tos {
		gpcm.KickPlayer(req.Pid, "banned")
	} else {
		gpcm.KickPlayer(req.Pid, "restricted")
	}

	ip := database.GetUserIP(pool, ctx, req.Pid)
	return mmss("result", "success", "ip", ip), http.StatusOK
}

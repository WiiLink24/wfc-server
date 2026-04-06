package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func HandleBan(w http.ResponseWriter, r *http.Request) {
	var success bool
	var err string
	var statusCode int

	switch r.Method {
	case http.MethodPost:
		success, err, statusCode = handleBanImpl(r)
	case http.MethodOptions:
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	default:
		err = "incorrect request. POST only."
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST")
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")

		if success {
			jsonData, _ = json.Marshal(map[string]string{"success": "true"})
		} else {
			jsonData, _ = json.Marshal(map[string]string{"error": err})
		}
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))

	w.WriteHeader(statusCode)
	_, _ = w.Write(jsonData)
}

type BanRequestSpec struct {
	Secret       string `json:"secret"`
	ProfileID    uint32 `json:"pid"`
	Days         uint64 `json:"days"`
	Hours        uint64 `json:"hours"`
	Minutes      uint64 `json:"minutes"`
	Tos          bool   `json:"tos"`
	Reason       string `json:"reason"`
	ReasonHidden string `json:"reason_hidden"`
	Moderator    string `json:"moderator"`
}

func handleBanImpl(r *http.Request) (bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, "Unable to read request body", http.StatusBadRequest
	}

	var req BanRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return false, "Invalid API secret in request", http.StatusUnauthorized
	}

	if req.ProfileID == 0 {
		return false, "Profile ID missing or 0 in request", http.StatusBadRequest
	}

	if req.Reason == "" {
		return false, "Missing ban reason in request", http.StatusBadRequest
	}

	moderator := req.Moderator
	if moderator == "" {
		moderator = "admin"
	}

	minutes := req.Days*24*60 + req.Hours*60 + req.Minutes
	if minutes == 0 {
		return false, "Ban length missing or 0", http.StatusBadRequest
	}

	length := time.Duration(minutes) * time.Minute

	logging.Notice("API:"+moderator, "Ban profile:", aurora.Cyan(req.ProfileID), "TOS:", aurora.Cyan(req.Tos), "Length:", aurora.Cyan(length), "Reason:", aurora.BrightCyan(req.Reason), "Reason (Hidden):", aurora.BrightCyan(req.ReasonHidden))

	if !db.BanUser(req.ProfileID, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		return false, "Failed to ban user", http.StatusInternalServerError
	}

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgProfileRestrictedCustom)

	logging.Event("profile_banned", map[string]any{
		"profile_id":     req.ProfileID,
		"tos_violation":  req.Tos,
		"length_minutes": minutes,
		"reason":         req.Reason,
		"reason_hidden":  req.ReasonHidden,
		"moderator":      moderator,
	})

	return true, "", http.StatusOK
}

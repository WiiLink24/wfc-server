package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
	"wwfc/database"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func HandleBan(w http.ResponseWriter, r *http.Request) {
	var user *database.User
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		user, statusCode, err = handleBanImpl(r)
	} else if r.Method == http.MethodOptions {
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	} else {
		err = ErrPostOnly
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST")
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	if user == nil {
		user = &database.User{}
	}

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(UserActionResponse{*user, err == nil, resolveError(err)})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
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

func handleBanImpl(r *http.Request) (*database.User, int, error) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, http.StatusBadRequest, ErrRequestBody
	}

	var req BanRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return nil, http.StatusUnauthorized, ErrInvalidSecret
	}

	if req.ProfileID == 0 {
		return nil, http.StatusBadRequest, ErrPIDMissing
	}

	if req.Reason == "" {
		return nil, http.StatusBadRequest, ErrReason
	}

	moderator := req.Moderator
	if moderator == "" {
		moderator = "admin"
	}

	minutes := req.Days*24*60 + req.Hours*60 + req.Minutes
	if minutes == 0 {
		return nil, http.StatusBadRequest, ErrLength
	}

	length := time.Duration(minutes) * time.Minute

	logging.Notice("API:"+moderator, "Ban profile:", aurora.Cyan(req.ProfileID), "TOS:", aurora.Cyan(req.Tos), "Length:", aurora.Cyan(length), "Reason:", aurora.BrightCyan(req.Reason), "Reason (Hidden):", aurora.BrightCyan(req.ReasonHidden))

	if !database.BanUser(pool, ctx, req.ProfileID, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		return nil, http.StatusInternalServerError, ErrTransaction
	}

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgProfileRestrictedCustom)

	user, success := database.GetProfile(pool, ctx, req.ProfileID)

	if !success {
		return nil, http.StatusInternalServerError, ErrUserQueryTransaction
	}

	return &user, http.StatusOK, nil
}

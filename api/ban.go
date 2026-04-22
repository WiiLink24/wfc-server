package api

import (
	"net/http"
	"time"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type BanRequestSpec struct {
	AuthInfo
	ProfileID    uint32 `json:"pid"`
	Days         uint64 `json:"days"`
	Hours        uint64 `json:"hours"`
	Minutes      uint64 `json:"minutes"`
	Tos          bool   `json:"tos"`
	Reason       string `json:"reason"`
	ReasonHidden string `json:"reason_hidden"`
	Moderator    string `json:"moderator"`
}

func HandleBan(w http.ResponseWriter, r *http.Request) {
	req := BanRequestSpec{}
	err := parsePost(r, w, &req, RoleModerator)
	if err != nil {
		return
	}

	if req.ProfileID == 0 {
		replyError(w, http.StatusBadRequest, APIErrorInvalidProfileID)
		return
	}

	if req.Reason == "" {
		replyError(w, http.StatusBadRequest, APIErrorInvalidBanReason)
		return
	}

	moderator := req.Moderator
	if moderator == "" {
		moderator = "admin"
	}

	minutes := req.Days*24*60 + req.Hours*60 + req.Minutes
	if minutes == 0 {
		replyError(w, http.StatusBadRequest, APIErrorInvalidBanLength)
		return
	}

	length := time.Duration(minutes) * time.Minute

	logging.Notice("API:"+moderator, "Ban profile:", aurora.Cyan(req.ProfileID), "TOS:", aurora.Cyan(req.Tos), "Length:", aurora.Cyan(length), "Reason:", aurora.BrightCyan(req.Reason), "Reason (Hidden):", aurora.BrightCyan(req.ReasonHidden))

	if !db.BanUser(req.ProfileID, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		replyError(w, http.StatusInternalServerError, APIErrorBanFailed)
		return
	}

	replyOK(w, nil)

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgProfileRestrictedCustom)

	logging.Event("profile_banned", map[string]any{
		"profile_id":     req.ProfileID,
		"tos_violation":  req.Tos,
		"length_minutes": minutes,
		"reason":         req.Reason,
		"reason_hidden":  req.ReasonHidden,
		"moderator":      moderator,
	})
}

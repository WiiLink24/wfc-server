package api

import (
	"net/http"
	"time"
	"wwfc/database"
	"wwfc/gpcm"
)

type BanRequest struct {
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

var BanRoute = MakeRouteSpec[BanRequest, UserActionResponse](
	true,
	"/api/ban",
	func(req any, v bool, _ *http.Request) (any, int, error) {
		return handleUserAction(req.(BanRequest), v, handleBanImpl)
	},
	http.MethodPost,
)

func handleBanImpl(req BanRequest, _ bool) (*database.User, int, error) {
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

	if !database.BanUser(pool, ctx, req.ProfileID, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		return nil, http.StatusInternalServerError, ErrTransaction
	}

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgProfileRestrictedCustom)

	user, err := database.GetProfile(pool, ctx, req.ProfileID)

	if err != nil {
		return nil, http.StatusInternalServerError, ErrUserQueryTransaction
	}

	return &user, http.StatusOK, nil
}

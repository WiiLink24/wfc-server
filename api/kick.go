package api

import (
	"net/http"
	"wwfc/database"
	"wwfc/gpcm"
)

type KickRequest struct {
	Secret    string `json:"secret"`
	Reason    string `json:"reason"`
	ProfileID uint32 `json:"pid"`
}

var KickRoute = MakeRouteSpec[KickRequest, UserActionResponse](
	true,
	"/api/kick",
	func(req any, v bool, _ *http.Request) (any, int, error) {
		return handleUserAction(req.(KickRequest), v, handleKickImpl)
	},
	http.MethodPost,
)

func handleKickImpl(req KickRequest, _ bool) (*database.User, int, error) {
	if req.ProfileID == 0 {
		return nil, http.StatusBadRequest, ErrPIDMissing
	}

	if req.Reason == "" {
		return nil, http.StatusBadRequest, ErrReason
	}

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgKickedCustom)

	user, err := database.GetProfile(pool, ctx, req.ProfileID)

	if err != nil {
		return nil, http.StatusInternalServerError, ErrUserQueryTransaction
	}

	return &user, http.StatusOK, nil
}

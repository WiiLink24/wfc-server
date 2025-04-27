package api

import (
	"net/http"
	"wwfc/database"
)

type UnbanRequest struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

var UnbanRoute = MakeRouteSpec[UnbanRequest, UserActionResponse](
	true,
	"/api/unban",
	func(req any, v bool, _ *http.Request) (any, int, error) {
		return handleUserAction(req.(UnbanRequest), v, handleUnbanImpl)
	},
	http.MethodPost,
)

func handleUnbanImpl(req UnbanRequest, _ bool) (*database.User, int, error) {
	if req.ProfileID == 0 {
		return nil, http.StatusBadRequest, ErrPIDMissing
	}

	if !database.UnbanUser(pool, ctx, req.ProfileID) {
		return nil, http.StatusInternalServerError, ErrTransaction
	}

	user, err := database.GetProfile(pool, ctx, req.ProfileID)

	if err != nil {
		return nil, http.StatusInternalServerError, ErrUserQueryTransaction
	}

	return &user, http.StatusOK, nil
}

package api

import (
	"net/http"
	"wwfc/database"
)

type ClearRequest struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

var ClearRoute = MakeRouteSpec[ClearRequest, UserActionResponse](
	true,
	"/api/clear",
	func(req any, v bool, _ *http.Request) (any, int, error) {
		return handleUserAction(req.(ClearRequest), v, handleClearImpl)
	},
	http.MethodPost,
)

func handleClearImpl(req ClearRequest, _ bool) (*database.User, int, error) {
	user, success := database.ClearProfile(pool, ctx, req.ProfileID)

	if !success {
		return nil, http.StatusInternalServerError, ErrTransaction
	}

	return &user, http.StatusOK, nil
}

package api

import (
	"net/http"
	"wwfc/database"
)

type RemoveHashRequest struct {
	Secret  string `json:"secret"`
	PackID  uint32 `json:"pack_id"`
	Version uint32 `json:"version"`
}

type RemoveHashResponse struct {
	Success bool
	Error   string
}

var RemoveHashRoute = MakeRouteSpec[RemoveHashRequest, RemoveHashResponse](
	true,
	"/api/remove_hash",
	HandleRemoveHash,
	http.MethodPost,
)

func HandleRemoveHash(req any, _ bool, r *http.Request) (any, int, error) {
	code := http.StatusOK
	_req := req.(RemoveHashRequest)
	err := database.RemoveHash(pool, ctx, _req.PackID, _req.Version)

	if err != nil {
		code = http.StatusInternalServerError
	}

	return RemoveHashResponse{}, code, err
}

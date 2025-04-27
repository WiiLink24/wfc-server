package api

import (
	"net/http"
	"wwfc/database"
)

type GetHashRequest struct {
	Secret string `json:"secret"`
}

type GetHashResponse struct {
	Success bool
	Error   string
	Hashes  database.HashStore
}

var GetHashRoute = MakeRouteSpec[GetHashRequest, GetHashResponse](
	true,
	"/api/get_hash",
	func(_ any, _ bool, _ *http.Request) (any, int, error) {
		res := GetHashResponse{}
		res.Hashes = database.GetHashes()

		return res, http.StatusOK, nil
	},
	http.MethodPost,
)

package api

import (
	"net/http"
	"wwfc/database"
)

type HashRequest struct {
	Secret    string `json:"secret"`
	PackID    uint32 `json:"pack_id"`
	Version   uint32 `json:"version"`
	HashNTSCU string `json:"hash_ntscu"`
	HashNTSCJ string `json:"hash_ntscj"`
	HashNTSCK string `json:"hash_ntsck"`
	HashPAL   string `json:"hash_pal"`
}

type HashResponse struct {
	Success bool
	Error   string
}

var SetHashRoute = MakeRouteSpec[HashRequest, HashResponse](
	true,
	"/api/set_hash",
	HandleSetHash,
	http.MethodPost,
)

func HandleSetHash(req any, _ bool, _ *http.Request) (any, int, error) {
	code := http.StatusOK
	_req := req.(HashRequest)

	err := database.UpdateHash(
		pool,
		ctx,
		_req.PackID,
		_req.Version,
		_req.HashNTSCU,
		_req.HashNTSCJ,
		_req.HashNTSCK,
		_req.HashPAL,
	)

	if err != nil {
		code = http.StatusInternalServerError
	}

	return HashResponse{}, code, err
}

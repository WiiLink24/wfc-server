package api

import (
	"net/http"
	"wwfc/gpcm"
)

type MotdRequest struct {
	Secret string `json:"secret"`
	Motd   string `json:"motd"`
}

type MotdResponse struct {
	Motd    string
	Success bool
	Error   string
}

var MotdRoute = MakeRouteSpec[MotdRequest, MotdResponse](
	false,
	"/api/motd",
	HandleMotd,
	http.MethodPost,
	http.MethodGet,
)

func HandleMotd(req any, v bool, r *http.Request) (any, int, error) {
	code := http.StatusOK
	var err error

	_req := req.(MotdRequest)
	res := MotdResponse{}

	if r.Method == http.MethodPost {
		if !v {
			return res, http.StatusForbidden, ErrInvalidSecret
		}

		err = gpcm.SetMessageOfTheDay(_req.Motd)
		res.Motd = _req.Motd
	} else if r.Method == http.MethodGet {
		res.Motd, err = gpcm.GetMessageOfTheDay()
	}

	if err != nil {
		code = http.StatusInternalServerError
	}

	return res, code, err
}

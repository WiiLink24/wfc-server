package api

import (
	"net/http"
	"time"
	"wwfc/database"
)

type PinfoRequest struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

var PinfoRoute = MakeRouteSpec[PinfoRequest, UserActionResponse](
	false,
	"/api/pinfo",
	func(req any, v bool, _ *http.Request) (any, int, error) {
		return handleUserAction(req.(PinfoRequest), v, handlePinfoImpl)
	},
	http.MethodPost,
)

func handlePinfoImpl(req PinfoRequest, validSecret bool) (*database.User, int, error) {
	realUser, err := database.GetProfile(pool, ctx, req.ProfileID)
	var ret *database.User

	if err != nil {
		if !validSecret {
			err = ErrUserQuery
		}

		return &database.User{}, http.StatusInternalServerError, err
	}

	if !validSecret {
		// Invalid secret, only report normal user info
		ret = &database.User{
			ProfileId:    realUser.ProfileId,
			Restricted:   realUser.Restricted,
			BanReason:    realUser.BanReason,
			OpenHost:     realUser.OpenHost,
			LastInGameSn: realUser.LastInGameSn,
			BanIssued:    realUser.BanIssued,
			BanExpires:   realUser.BanExpires,
		}
	} else {
		ret = &realUser
	}

	_, offset := time.Now().Zone()

	// Add the offset to the time and then convert it back to local.
	// The DB stores times in the server's locale but they are unmarshaled as
	// UTC. This corrects for that
	if ret.BanIssued != nil {
		fixedIssued := ret.BanIssued.Add(time.Duration(-offset) * time.Second).Local()
		ret.BanIssued = &fixedIssued
	}

	if ret.BanExpires != nil {
		fixedExpires := ret.BanExpires.Add(time.Duration(-offset) * time.Second).Local()
		ret.BanExpires = &fixedExpires
	}

	return ret, http.StatusOK, nil
}

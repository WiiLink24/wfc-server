package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
	"wwfc/database"
)

func HandlePinfo(w http.ResponseWriter, r *http.Request) {
	var user *database.User
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		user, statusCode, err = handlePinfoImpl(r)
	} else if r.Method == http.MethodOptions {
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	} else {
		err = ErrPostOnly
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST")
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	if user == nil {
		user = &database.User{}
	}

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(UserActionResponse{*user, err == nil, resolveError(err)})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type PinfoRequestSpec struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

func handlePinfoImpl(r *http.Request) (*database.User, int, error) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, http.StatusBadRequest, ErrRequestBody
	}

	var req PinfoRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, http.StatusBadRequest, ErrRequestBody
	}

	realUser, success := database.GetProfile(pool, ctx, req.ProfileID)
	var ret *database.User

	if !success {
		return &database.User{}, http.StatusInternalServerError, ErrUserQuery
	}

	if apiSecret == "" || req.Secret != apiSecret {
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
		fixedExpires := ret.BanIssued.Add(time.Duration(-offset) * time.Second).Local()
		ret.BanExpires = &fixedExpires
	}

	return ret, http.StatusOK, nil
}

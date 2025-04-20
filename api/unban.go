package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

func HandleUnban(w http.ResponseWriter, r *http.Request) {
	var user *database.User
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		user, statusCode, err = handleUnbanImpl(r)
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

type UnbanRequestSpec struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

func handleUnbanImpl(r *http.Request) (*database.User, int, error) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	var req UnbanRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return nil, http.StatusUnauthorized, ErrInvalidSecret
	}

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

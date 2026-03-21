package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

type PinfoRequestSpec struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

type PinfoResponse struct {
	User    database.User `json:"User"`
	Success bool          `json:"Success"`
	Error   string        `json:"Error"`
}

func HandlePinfo(w http.ResponseWriter, r *http.Request) {
	var response PinfoResponse
	var statusCode int

	if r.Method == http.MethodPost {
		response, statusCode = handlePinfoImpl(r)
	} else if r.Method == http.MethodOptions {
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	} else {
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST")
		response = PinfoResponse{
			Success: false,
			Error:   "Incorrect request. POST only.",
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte
	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(response)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

func handlePinfoImpl(r *http.Request) (PinfoResponse, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return PinfoResponse{
			Success: false,
			Error:   "Unable to read request body",
		}, http.StatusBadRequest
	}

	var req PinfoRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return PinfoResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusBadRequest
	}

	if req.ProfileID == 0 {
		return PinfoResponse{
			Success: false,
			Error:   "Profile ID missing or 0 in request",
		}, http.StatusBadRequest
	}

	realUser, ok := database.GetProfile(pool, ctx, req.ProfileID)
	if !ok {
		return PinfoResponse{
			User:    database.User{},
			Success: false,
			Error:   "Failed to find user in the database",
		}, http.StatusInternalServerError
	}

	user := realUser
	if apiSecret == "" || req.Secret != apiSecret {
		// Invalid or missing secret: return only the public-safe subset.
		user = database.User{
			ProfileId:    realUser.ProfileId,
			Restricted:   realUser.Restricted,
			BanReason:    realUser.BanReason,
			OpenHost:     realUser.OpenHost,
			LastInGameSn: realUser.LastInGameSn,
		}
	}

	return PinfoResponse{
		User:    user,
		Success: true,
		Error:   "",
	}, http.StatusOK
}

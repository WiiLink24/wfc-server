package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/gpcm"
)

func HandleKick(w http.ResponseWriter, r *http.Request) {
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		success, err, statusCode = handleKickImpl(r)
	} else if r.Method == http.MethodOptions {
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	} else {
		err = "Incorrect request. POST only."
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST")
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")

		if success {
			jsonData, _ = json.Marshal(map[string]string{"success": "true"})
		} else {
			jsonData, _ = json.Marshal(map[string]string{"error": err})
		}
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))

	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type KickRequestSpec struct {
	Secret    string `json:"secret"`
	Reason    string `json:"reason"`
	ProfileID uint32 `json:"pid"`
}

func handleKickImpl(r *http.Request) (bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, "Unable to read request body", http.StatusBadRequest
	}

	var req KickRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return false, "Invalid API secret in request", http.StatusUnauthorized
	}

	if req.ProfileID == 0 {
		return false, "Profile ID missing or 0 in request", http.StatusBadRequest
	}

	if req.Reason == "" {
		return false, "Missing kick reason in request", http.StatusBadRequest
	}

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgKickedCustom)

	return true, "", http.StatusOK
}

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
	"wwfc/database"
	"wwfc/gpcm"
)

func HandleBan(w http.ResponseWriter, r *http.Request) {
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		success, err, statusCode = handleBanImpl(r)
	} else {
		err = "Incorrect request. POST only."
		statusCode = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte
	if success {
		jsonData, _ = json.Marshal(map[string]string{"success": "true"})
	} else {
		jsonData, _ = json.Marshal(map[string]string{"error": err})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type BanRequestSpec struct {
	Secret       string
	Pid          uint32
	Days         uint64
	Hours        uint64
	Minutes      uint64
	Tos          bool
	Reason       string
	ReasonHidden string
	Moderator    string
}

func handleBanImpl(r *http.Request) (bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, "Unable to read request body", http.StatusBadRequest
	}

	var req BanRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return false, "Invalid API secret in request", http.StatusUnauthorized
	}

	if req.Pid == 0 {
		return false, "pid missing or 0 in request", http.StatusBadRequest
	}

	if req.Reason == "" {
		return false, "Missing ban reason in request", http.StatusBadRequest
	}

	moderator := req.Moderator
	if moderator == "" {
		moderator = "admin"
	}

	minutes := req.Days*24*60 + req.Hours*60 + req.Minutes
	if minutes == 0 {
		return false, "Ban length missing or 0", http.StatusBadRequest
	}

	length := time.Duration(minutes) * time.Minute

	if !database.BanUser(pool, ctx, req.Pid, req.Tos, length, req.Reason, req.ReasonHidden, moderator) {
		return false, "Failed to ban user", http.StatusInternalServerError
	}

	gpcm.KickPlayerCustomMessage(req.Pid, req.Reason, gpcm.WWFCErrorMessage{
		ErrorCode: 22002,
		MessageRMC: map[byte]string{
			gpcm.LangEnglish: "" +
				"You have been banned from WiiLink WFC\n" +
				"Reason: " + req.Reason + "\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
		},
	})

	return true, "", http.StatusOK
}

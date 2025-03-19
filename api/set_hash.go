package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func HandleSetHash(w http.ResponseWriter, r *http.Request) {
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		success, err, statusCode = handleSetHashImpl(r)
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
		jsonData, _ = json.Marshal(HashResponse{success, err})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type HashRequestSpec struct {
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

func handleSetHashImpl(r *http.Request) (bool, string, int) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, "Unable to read request body", http.StatusBadRequest
	}

	var req HashRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return false, "Invalid API secret in request", http.StatusUnauthorized
	}

	logging.Notice("API", "Hashes Received, PackID:", aurora.Cyan(req.PackID), "Version:", aurora.Cyan(req.Version), "\nNTSCU:", aurora.Cyan(req.HashNTSCU), "\nNTSCJ:", aurora.Cyan(req.HashNTSCJ), "\nNTSCK:", aurora.Cyan(req.HashNTSCK), "\nPAL:", aurora.Cyan(req.HashPAL))

	err = database.UpdateHash(pool, ctx, req.PackID, req.Version, req.HashNTSCU, req.HashNTSCJ, req.HashNTSCK, req.HashPAL)
	if err != nil {
		return false, err.Error(), http.StatusInternalServerError
	}

	return true, "", http.StatusOK
}

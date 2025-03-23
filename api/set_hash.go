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
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		statusCode, err = handleSetHashImpl(r)
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

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(HashResponse{err == nil, resolveError(err)})
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

func handleSetHashImpl(r *http.Request) (int, error) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, ErrRequestBody
	}

	var req HashRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return http.StatusBadRequest, ErrInvalidSecret
	}

	logging.Notice("API", "Hashes Received, PackID:", aurora.Cyan(req.PackID), "Version:", aurora.Cyan(req.Version), "\nNTSCU:", aurora.Cyan(req.HashNTSCU), "\nNTSCJ:", aurora.Cyan(req.HashNTSCJ), "\nNTSCK:", aurora.Cyan(req.HashNTSCK), "\nPAL:", aurora.Cyan(req.HashPAL))

	err = database.UpdateHash(pool, ctx, req.PackID, req.Version, req.HashNTSCU, req.HashNTSCJ, req.HashNTSCK, req.HashPAL)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

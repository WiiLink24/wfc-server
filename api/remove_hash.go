package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

func HandleRemoveHash(w http.ResponseWriter, r *http.Request) {
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		success, err, statusCode = handleRemoveHashImpl(r)
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
		jsonData, _ = json.Marshal(RemoveHashResponse{success, err})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type RemoveHashRequestSpec struct {
	Secret  string `json:"secret"`
	PackID  uint32 `json:"pack_id"`
	Version uint32 `json:"version"`
}

type RemoveHashResponse struct {
	Success bool
	Error   string
}

func handleRemoveHashImpl(r *http.Request) (bool, string, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, "Unable to read request body", http.StatusBadRequest
	}

	var req RemoveHashRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return false, "Invalid API secret in request", http.StatusUnauthorized
	}

	err = database.RemoveHash(pool, ctx, req.PackID, req.Version)
	if err != nil {
		return false, err.Error(), http.StatusInternalServerError
	}

	return true, "", http.StatusOK
}

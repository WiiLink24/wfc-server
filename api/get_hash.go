package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

func HandleGetHash(w http.ResponseWriter, r *http.Request) {
	var store database.HashStore
	var success bool
	var err string
	var statusCode int

	if r.Method == http.MethodPost {
		store, success, err, statusCode = handleGetHashImpl(r)
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
		jsonData, _ = json.Marshal(GetHashResponse{success, err, store})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type GetHashRequestSpec struct {
	Secret string `json:"secret"`
}

type GetHashResponse struct {
	Success bool
	Error   string
	Hashes  database.HashStore
}

func handleGetHashImpl(r *http.Request) (database.HashStore, bool, string, int) {
	ret := database.HashStore{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ret, false, "Unable to read request body", http.StatusBadRequest
	}

	var req GetHashRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return ret, false, err.Error(), http.StatusBadRequest
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return nil, false, "Invalid API secret in request", http.StatusUnauthorized
	}

	return database.GetHashes(), true, "", http.StatusOK
}

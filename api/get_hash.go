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
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		store, statusCode, err = handleGetHashImpl(r)
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
		jsonData, _ = json.Marshal(GetHashResponse{err == nil, resolveError(err), store})
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

func handleGetHashImpl(r *http.Request) (database.HashStore, int, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, http.StatusBadRequest, ErrRequestBody
	}

	var req GetHashRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return nil, http.StatusUnauthorized, ErrInvalidSecret
	}

	return database.GetHashes(), http.StatusOK, nil
}

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/database"
)

func HandleRemoveHash(w http.ResponseWriter, r *http.Request) {
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		statusCode, err = handleRemoveHashImpl(r)
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
		jsonData, _ = json.Marshal(RemoveHashResponse{err == nil, resolveError(err)})
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

func handleRemoveHashImpl(r *http.Request) (int, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, ErrRequestBody
	}

	var req RemoveHashRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return http.StatusBadRequest, ErrInvalidSecret
	}

	err = database.RemoveHash(pool, ctx, req.PackID, req.Version)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/gpcm"
)

func HandleMotd(w http.ResponseWriter, r *http.Request) {
	var motd string
	var statusCode int
	var err error

	if r.Method == http.MethodPost {
		statusCode, err = handleMotdImpl(r)
	} else if r.Method == http.MethodGet {
		_motd, motdErr := gpcm.GetMessageOfTheDay()
		if motdErr != nil {
			err = motdErr
			statusCode = http.StatusInternalServerError
		} else {
			motd = _motd
			statusCode = http.StatusOK
		}
	} else if r.Method == http.MethodOptions {
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	} else {
		err = ErrPostGetOnly
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST, GET")
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte

	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(MotdResponse{motd, err == nil, resolveError(err)})
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}

type MotdRequestSpec struct {
	Secret string `json:"secret"`
	Motd   string `json:"motd"`
}

type MotdResponse struct {
	Motd    string
	Success bool
	Error   string
}

func handleMotdImpl(r *http.Request) (int, error) {
	// TODO: Actual authentication rather than a fixed secret

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, ErrRequestBody
	}

	var req MotdRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if apiSecret == "" || req.Secret != apiSecret {
		return http.StatusUnauthorized, ErrInvalidSecret
	}

	err = gpcm.SetMessageOfTheDay(req.Motd)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

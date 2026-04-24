package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

type APIErrorString string

const (
	APIErrorFailedAuthentication APIErrorString = "failed_authentication"
	APIErrorInvalidQuery         APIErrorString = "invalid_query"
	APIErrorInvalidProfileID     APIErrorString = "invalid_profile_id"
	APIErrorInvalidReason        APIErrorString = "invalid_reason"
	APIErrorInvalidBanLength     APIErrorString = "invalid_ban_length"
	APIErrorBanFailed            APIErrorString = "ban_failed"
	APIErrorUnbanFailed          APIErrorString = "unban_failed"
	APIErrorBanNotFound          APIErrorString = "ban_not_found"
)

type APIError struct {
	Error string `json:"error"`
}

type Role string

// Values currently just made up
const (
	RoleNone      Role = "none" // Not signed in
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
)

type AuthInfo struct {
	Secret string `json:"secret"`
}

var (
	errOptionsRequest  = errors.New("OPTIONS request")
	errIncorrectMethod = errors.New("incorrect HTTP method")
	errNoAuthInfo      = errors.New("request struct does not contain AuthInfo fields")
	errAuthFailed      = errors.New("authentication failed")
)

func parseGet(r *http.Request, w http.ResponseWriter, requiredRole Role) (query url.Values, err error) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch {
	case r.Method == http.MethodGet:
		break

	case r.Method == http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return nil, errOptionsRequest

	default:
		w.Header().Set("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, errIncorrectMethod
	}

	query, err = url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	if requiredRole == RoleNone {
		return query, nil
	}

	authInfo := makeAuthInfo(query)
	if !authenticate(authInfo, requiredRole) {
		replyError(w, http.StatusUnauthorized, APIErrorFailedAuthentication)
		return nil, errAuthFailed
	}
	return query, nil
}

func parsePost(r *http.Request, w http.ResponseWriter, parsed any, requiredRole Role) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch {
	case r.Method == http.MethodPost:
		break

	case r.Method == http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return errOptionsRequest

	default:
		w.Header().Set("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return errIncorrectMethod
	}

	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}

	err = json.Unmarshal(jsonData, parsed)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}

	if requiredRole == RoleNone {
		return nil
	}

	authInfo, ok := reflect.ValueOf(parsed).Elem().FieldByName("AuthInfo").Interface().(AuthInfo)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return errNoAuthInfo
	}
	if !authenticate(authInfo, requiredRole) {
		replyError(w, http.StatusUnauthorized, APIErrorFailedAuthentication)
		return errAuthFailed
	}
	return nil
}

func makeAuthInfo(query url.Values) AuthInfo {
	return AuthInfo{
		Secret: query.Get("secret"),
	}
}

func authenticate(authInfo AuthInfo, requiredRole Role) bool {
	return requiredRole == RoleNone || authInfo.Secret == apiSecret
}

func replyError(w http.ResponseWriter, statusCode int, errMsg APIErrorString) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	jsonData := []byte(`{"error":"` + string(errMsg) + `"}`)
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	_, _ = w.Write(jsonData)
}

func replyOK(w http.ResponseWriter, data any) {
	if data == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var jsonData []byte
	if reflect.ValueOf(data).Kind() == reflect.String {
		// Assume it's already JSON
		jsonData = []byte(data.(string))
	} else {
		var err error
		jsonData, err = json.Marshal(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	_, _ = w.Write(jsonData)
}

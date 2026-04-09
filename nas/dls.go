package nas

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var dlsActions = map[string]func(moduleName string, fields map[string][]byte) []byte{
	"count": dlsCount,
}

func handleDownloadEndpoint(w http.ResponseWriter, r *http.Request) {
	moduleName := getModuleName(r)

	fields, err := parseAuthRequest(r)
	if err != nil {
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	action := string(fields["action"])
	if action == "" {
		logging.Error(moduleName, "No action in form")
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	rhgamecd, ok := fields["rhgamecd"]
	if !ok || !isValidRHGameCode(string(rhgamecd)) {
		logging.Error(moduleName, "Missing or invalid rhgamecd")
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	if actionFunc, exists := dlsActions[strings.ToLower(action)]; exists {
		reply := actionFunc(moduleName, fields)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(len(reply)))
		_, err := w.Write(reply)
		if err != nil {
			logging.Error(moduleName, "Error writing response:", err)
		}
		return
	}

	logging.Error(moduleName, "Unknown action:", aurora.Cyan(action))
	replyHTTPError(w, 400, "400 Bad Request")
}

func dlsCount(moduleName string, fields map[string][]byte) []byte {
	dlcFolder := filepath.Join(dlcDir, string(fields["rhgamecd"]))

	dir, err := os.ReadDir(dlcFolder)
	if err != nil {
		return []byte{'0', 0}
	}

	return append([]byte(strconv.Itoa(len(dir))), 0)
}

func isValidRHGameCode(rhgamecd string) bool {
	if len(rhgamecd) != 4 {
		return false
	}

	return common.IsUppercaseAlphanumeric(rhgamecd)
}

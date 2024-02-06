package gamestats

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func HandleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	logging.Info("GSTATS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))

	u, err := url.Parse(r.URL.String())
	if err != nil {
		replyHTTPError(w, http.StatusBadRequest, "400 Bad Request")
		return
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		replyHTTPError(w, http.StatusBadRequest, "400 Bad Request")
		return
	}

	path := u.Path
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	gameName := path
	subPath := ""
	slashIndex := strings.Index(gameName, "/")
	if slashIndex != -1 {
		gameName = gameName[:slashIndex]
		subPath = path[slashIndex:]
	}

	game := common.GetGameInfoByName(gameName)
	if game == nil {
		replyHTTPError(w, http.StatusNotFound, "404 Not Found")
		return
	}

	hash := query.Get("hash")
	var data string

	if hash == "" {
		// No hash, just return token
		data = common.RandomString(32)
	} else {
		// TODO: Handle subPath to get data here
		common.UNUSED(subPath)
		data = ""

		if game.GameStatsVersion > 1 {
			// SHA-1 hash GamestatsKey + base64(data) + GameStatsKey
			hashData := game.GameStatsKey + base64.URLEncoding.EncodeToString([]byte(data)) + game.GameStatsKey
			hasher := sha1.New()
			hasher.Write([]byte(hashData))
			// Append the hash sum as a hex string
			data += hex.EncodeToString(hasher.Sum(nil))
		}
	}

	w.Header().Set("Content-type", "text/html")
	w.Header().Set("Server", "Microsoft-IIS/6.0")
	w.Header().Add("Server", "GSTPRDSTATSWEB2")
	w.Header().Set("X-Powered-By", "ASP.NET")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(200)
	w.Write([]byte(data))
}

func replyHTTPError(w http.ResponseWriter, errorCode int, errorString string) {
	response := "<html>\n" +
		"<head><title>" + errorString + "</title></head>\n" +
		"<body>\n" +
		"<center><h1>" + errorString + "</h1></center>\n" +
		"<hr><center>" + serverName + "</center>\n" +
		"</body>\n" +
		"</html>\n"

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Connection", "close")
	w.WriteHeader(errorCode)
	w.Write([]byte(response))
}

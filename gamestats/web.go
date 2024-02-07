package gamestats

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func HandleWebRequest(w http.ResponseWriter, r *http.Request) {
	logging.Info("GSTATS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
	moduleName := "GSTATS:" + r.RemoteAddr

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

	var response []byte

	hash := query.Get("hash")
	token := calculateToken(r.URL, r.Host)

	if hash == "" {
		// No hash, just return token
		response = []byte(token)
	} else {
		// Check hash supplied by client
		hasher := sha1.New()
		hasher.Write([]byte(game.GameStatsKey))
		hasher.Write([]byte(token))
		expectedHash := hex.EncodeToString(hasher.Sum(nil))

		if hash != expectedHash {
			logging.Warn(moduleName, "Invalid hash")
		}

		switch subPath {
		case "/web/client/get2.asp":
			response = handleGet2(game, query)

		default:
			logging.Warn(moduleName, "Unhandled path:", aurora.Cyan(subPath))
		}

		// Padding to appease DWC
		response = append(response, make([]byte, 13)...)

		if game.GameStatsVersion > 1 {
			// SHA-1 hash GamestatsKey + base64(data) + GameStatsKey
			hashData := game.GameStatsKey + base64.URLEncoding.EncodeToString([]byte(response)) + game.GameStatsKey
			hasher := sha1.New()
			hasher.Write([]byte(hashData))
			// Append the hash sum as a hex string
			response = append(response, []byte(hex.EncodeToString(hasher.Sum(nil)))...)
		}
	}

	w.Header().Set("Content-type", "text/html")
	w.Header().Set("Server", "Microsoft-IIS/6.0")
	w.Header().Add("Server", "GSTPRDSTATSWEB2")
	w.Header().Set("X-Powered-By", "ASP.NET")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.WriteHeader(200)
	w.Write(response)
}

func calculateToken(u *url.URL, host string) string {
	newURL := *u
	newURL.RawQuery = url.Values{
		"pid": {u.Query().Get("pid")},
	}.Encode()

	hasher := sha256.New()
	hasher.Write([]byte(host + newURL.String()))
	hasher.Write([]byte(webSalt))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))[:32]
}

func handleGet2(game *common.GameInfo, query url.Values) []byte {
	data := binary.LittleEndian.AppendUint32([]byte{}, 1) // RNK_GET
	data = binary.LittleEndian.AppendUint32(data, 0)      // count
	return data
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

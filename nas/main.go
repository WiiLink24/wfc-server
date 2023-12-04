package nas

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/nhttp"
	"wwfc/sake"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	if err != nil {
		panic(err)
	}

	address := config.Address + ":" + config.Port

	logging.Notice("NAS", "Starting HTTP server on", address)
	panic(nhttp.ListenAndServe(address, http.HandlerFunc(handleRequest)))
}

var regexSakeHost = regexp.MustCompile(`^([a-z\-]+\.)?sake\.gs\.`)
var regexStage1URL = regexp.MustCompile(`^/w([0-9])$`)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// TODO: Move this to its own server
	// Check for *.sake.gs.* or sake.gs.*
	if regexSakeHost.MatchString(r.Host) {
		// Redirect to the sake server
		sake.HandleRequest(w, r)
		return
	}

	logging.Notice("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
	moduleName := "NAS:" + r.RemoteAddr

	if r.URL.String() == "/ac" || r.URL.String() == "/pr" || r.URL.String() == "/download" {
		handleAuthRequest(moduleName, w, r)
		return
	}

	// TODO: Move this to its own server
	// Check for /payload
	if strings.HasPrefix(r.URL.String(), "/payload?") {
		handlePayloadRequest(moduleName, w, r)
		return
	}

	// Check for /online
	if r.URL.String() == "/online" {
		returnOnlineStats(w)
		return
	}

	// Stage 1
	if match := regexStage1URL.FindStringSubmatch(r.URL.String()); match != nil {
		val, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}

		downloadStage1(moduleName, w, r, val)
		return
	}

	replyHTTPError(w, 404, "404 Not Found")
}

func replyHTTPError(w http.ResponseWriter, errorCode int, errorString string) {
	response := "<html>\n"
	response += "<head><title>" + errorString + "</title></head>\n"
	response += "<body>\n"
	response += "<center><h1>" + errorString + "</h1></center>\n"
	response += "<hr><center>WiiLink</center>\n"
	response += "</body>\n"
	response += "</html>\n"

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.WriteHeader(errorCode)
	w.Write([]byte(response))
}

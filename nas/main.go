package nas

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"wwfc/api"
	"wwfc/common"
	"wwfc/gamestats"
	"wwfc/logging"
	"wwfc/nhttp"
	"wwfc/race"
	"wwfc/sake"

	"github.com/logrusorgru/aurora/v3"
)

var (
	serverName string
	server     *nhttp.Server
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	serverName = config.ServerName

	address := *config.NASAddress + ":" + config.NASPort

	if config.EnableHTTPS {
		go startHTTPSProxy(config)
	}

	err := CacheProfanityFile()
	if err != nil {
		logging.Info("NAS", err)
	}

	server = &nhttp.Server{
		Addr:        address,
		Handler:     http.HandlerFunc(handleRequest),
		IdleTimeout: 20 * time.Second,
		ReadTimeout: 10 * time.Second,
	}

	go func() {
		logging.Notice("NAS", "Starting HTTP server on", aurora.BrightCyan(address))

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, nhttp.ErrServerClosed) {
			panic(err)
		}
	}()
}

func Shutdown() {
	if server == nil {
		return
	}

	ctx, release := context.WithTimeout(context.Background(), 10*time.Second)
	defer release()

	err := server.Shutdown(ctx)
	if err != nil {
		logging.Error("NAS", "Error on HTTP shutdown:", err)
	}
}

var regexRaceHost = regexp.MustCompile(`^([a-z\-]+\.)?race\.gs\.`)
var regexSakeHost = regexp.MustCompile(`^([a-z\-]+\.)?sake\.gs\.`)
var regexGamestatsHost = regexp.MustCompile(`^([a-z\-]+\.)?gamestats2?\.gs\.`)
var regexStage1URL = regexp.MustCompile(`^/w([0-9])$`)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check for *.sake.gs.* or sake.gs.*
	if regexSakeHost.MatchString(r.Host) {
		// Redirect to the sake server
		sake.HandleRequest(w, r)
		return
	}

	// Check for *.gamestats(2).gs.* or gamestats(2).gs.*
	if regexGamestatsHost.MatchString(r.Host) {
		// Redirect to the gamestats server
		gamestats.HandleWebRequest(w, r)
		return
	}

	// Check for *.race.gs.* or race.gs.*
	if regexRaceHost.MatchString(r.Host) {
		// Redirect to the race server
		race.HandleRequest(w, r)
		return
	}

	moduleName := "NAS:" + r.RemoteAddr

	// Handle conntest server
	if strings.HasPrefix(r.Host, "conntest.") {
		handleConnectionTest(w)
		return
	}

	// Handle DWC auth requests
	if r.URL.String() == "/ac" || r.URL.String() == "/pr" || r.URL.String() == "/download" {
		handleAuthRequest(moduleName, w, r)
		return
	}

	// Handle /nastest.jsp
	if r.URL.Path == "/nastest.jsp" {
		handleNASTest(w)
		return
	}

	// Check for /payload
	if strings.HasPrefix(r.URL.String(), "/payload") {
		logging.Info("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
		handlePayloadRequest(moduleName, w, r)
		return
	}

	// Stage 1
	if match := regexStage1URL.FindStringSubmatch(r.URL.String()); match != nil {
		val, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}

		logging.Info("NAS", "Get stage 1:", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
		downloadStage1(w, val)
		return
	}

	// Check for /api/groups
	if r.URL.Path == "/api/groups" {
		api.HandleGroups(w, r)
		return
	}

	// Check for /api/stats
	if r.URL.Path == "/api/stats" {
		api.HandleStats(w, r)
		return
	}

	// Check for /api/ban
	if r.URL.Path == "/api/ban" {
		api.HandleBan(w, r)
		return
	}

	// Check for /api/unban
	if r.URL.Path == "/api/unban" {
		api.HandleUnban(w, r)
		return
	}

	// Check for /api/kick
	if r.URL.Path == "/api/kick" {
		api.HandleKick(w, r)
		return
	}

	if r.URL.Path == "/api/motd" {
		api.HandleMotd(w, r)
		return
	}

	if r.URL.Path == "/api/clear" {
		api.HandleClear(w, r)
		return
	}

	if r.URL.Path == "/api/hash" {
		api.HandleHash(w, r)
		return
	}

	logging.Info("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
	replyHTTPError(w, 404, "404 Not Found")
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
	w.Header().Set("Server", "Nintendo")
	w.WriteHeader(errorCode)
	w.Write([]byte(response))
}

func handleNASTest(w http.ResponseWriter) {
	response := "" +
		"<html>\n" +
		"<body>\n" +
		"</br>AuthServer is up</br> \n" +
		"\n" +
		"</body>\n" +
		"</html>\n"

	w.Header().Set("Content-Type", "text/html;charset=ISO-8859-1")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Connection", "close")
	w.Header().Set("NODE", "authserver-service.authserver.svc.cluster.local")
	w.Header().Set("Server", "Nintendo")

	w.WriteHeader(200)
	w.Write([]byte(response))
}

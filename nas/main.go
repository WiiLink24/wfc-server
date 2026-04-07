package nas

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"wwfc/api"
	"wwfc/common"
	"wwfc/gamestats"
	"wwfc/logging"
	"wwfc/race"
	"wwfc/sake"

	"github.com/logrusorgru/aurora/v3"
)

var (
	serverName           string
	server               *http.Server
	payloadServerAddress string
)

var (
	authMux      = http.NewServeMux()
	dlsMux       = http.NewServeMux()
	sakeMux      = http.NewServeMux()
	gamestatsMux = http.NewServeMux()
	raceMux      = http.NewServeMux()
)

var hostMuxes = map[*regexp.Regexp]*http.ServeMux{
	regexp.MustCompile(`^(nas|naswii)\.`):                         authMux,
	regexp.MustCompile(`^dls1\.`):                                 dlsMux,
	regexp.MustCompile(`(\.|^)gamestats2?\.(gs\.|gamespy\.com$)`): gamestatsMux,
	regexp.MustCompile(`(\.|^)sake\.(gs\.|gamespy\.com$)`):        sakeMux,
	regexp.MustCompile(`(\.|^)race\.(gs\.|gamespy\.com$)`):        raceMux,
}

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()
	serverName = config.ServerName
	address := *config.NASAddress + ":" + config.NASPort
	payloadServerAddress = config.PayloadServerAddress

	if config.EnableHTTPS {
		go startHTTPSProxy(config)
	}

	err := CacheProfanityFile()
	if err != nil {
		logging.Info("NAS", err)
	}

	server = &http.Server{
		Addr:        address,
		Handler:     http.HandlerFunc(handleRequest),
		IdleTimeout: 20 * time.Second,
		ReadTimeout: 10 * time.Second,
	}

	authMux.HandleFunc("/ac", handleAuthAccountEndpoint)
	authMux.HandleFunc("/pr", handleAuthProfanityEndpoint)

	dlsMux.HandleFunc("/download", handleDownloadEndpoint)

	if payloadServerAddress != "" {
		// Forward the request to the payload server
		authMux.HandleFunc("/payload", forwardPayloadRequest)
	} else {
		authMux.HandleFunc("/payload", handlePayloadRequest)
	}

	for i := 0; i <= 9; i++ {
		authMux.HandleFunc("/w"+strconv.Itoa(i), downloadStage1)
	}

	authMux.HandleFunc("/nastest.jsp", handleNASTest)

	http.HandleFunc("GET conntest.nintendowifi.net/", handleConnectionTest)

	api.RegisterHandlers(http.DefaultServeMux)
	sake.RegisterHandlers(sakeMux)
	race.RegisterHandlers(raceMux)
	gamestatsMux.HandleFunc("/", gamestats.HandleWebRequest)

	http.HandleFunc("/", handleUnknown)
	authMux.HandleFunc("/", handleUnknown)
	sakeMux.HandleFunc("/", handleUnknown)
	raceMux.HandleFunc("/", handleUnknown)

	go listenAndServe(address)
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

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check for host-specific muxes
	for regex, mux := range hostMuxes {
		if regex.MatchString(r.Host) {
			mux.ServeHTTP(w, r)
			return
		}
	}

	http.DefaultServeMux.ServeHTTP(w, r)
}

func getModuleName(r *http.Request) string {
	return "NAS:" + r.RemoteAddr
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
	_, _ = w.Write([]byte(response))
}

func handleUnknown(w http.ResponseWriter, r *http.Request) {
	logging.Info(getModuleName(r), "Unknown request:", aurora.Yellow(r.Method), aurora.Cyan(r.Host+r.URL.Path))
	replyHTTPError(w, http.StatusNotFound, "404 Not Found")
}

func handleNASTest(w http.ResponseWriter, r *http.Request) {
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
	_, _ = w.Write([]byte(response))
}

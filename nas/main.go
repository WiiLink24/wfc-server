package nas

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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

type nasListener struct {
	net.Listener
}

type nasConn struct {
	net.Conn
	reader io.Reader
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

	logging.Notice("NAS", "Starting HTTP server on", aurora.BrightCyan(address))

	l, err := net.Listen("tcp", address)
	common.ShouldNotError(err)
	listener := &nasListener{Listener: l}

	go func(l net.Listener) {
		defer func() {
			common.ShouldNotError(listener.Close())
		}()

		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}(listener)
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

func (l *nasListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &nasConn{
		Conn:   conn,
		reader: filterDuplicateHost(conn),
	}, nil
}

func (c *nasConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

// filterDuplicateHost wraps a net.Conn and filters out duplicate Host headers from the HTTP request,
// making the invalid requests sent by DWC acceptable to the standard library's HTTP server.
func filterDuplicateHost(c net.Conn) io.Reader {
	r := bufio.NewReader(c)

	// Read the first line of the HTTP request
	line, err := r.ReadString('\n')
	if err != nil {
		return io.MultiReader(strings.NewReader(line), r)
	}
	// Is this an HTTP request?
	if !strings.HasSuffix(line, "HTTP/1.1\r\n") {
		return io.MultiReader(strings.NewReader(line), r)
	}

	// Iterate through the HTTP headers and remove any duplicate Host headers
	var headers bytes.Buffer
	hostSeen := false
	for {
		headerLine, err := r.ReadString('\n')
		if err != nil || headerLine == "\r\n" {
			headers.WriteString(headerLine)
			break
		}
		if strings.HasPrefix(strings.ToLower(headerLine), "host:") {
			if hostSeen {
				continue
			}
			hostSeen = true
		}
		headers.WriteString(headerLine)
	}

	return io.MultiReader(strings.NewReader(line+headers.String()), r)
}

var regexRaceHost = regexp.MustCompile(`(\.|^)race\.(gs\.|gamespy\.com$)`)
var regexSakeHost = regexp.MustCompile(`(\.|^)sake\.(gs\.|gamespy\.com$)`)
var regexGamestatsHost = regexp.MustCompile(`(\.|^)gamestats2?\.(gs\.|gamespy\.com$)`)
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
		if payloadServerAddress != "" {
			// Forward the request to the payload server
			forwardPayloadRequest(moduleName, w, r)
		} else {
			handlePayloadRequest(moduleName, w, r)
		}
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
	_, _ = w.Write([]byte(response))
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
	_, _ = w.Write([]byte(response))
}

func forwardPayloadRequest(moduleName string, w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	r.URL.Scheme = "http"
	r.URL.Host = payloadServerAddress
	r.RequestURI = ""
	r.Host = payloadServerAddress

	resp, err := client.Do(r)
	if err != nil {
		logging.Error(moduleName, "Error forwarding payload request:", err)
		replyHTTPError(w, http.StatusBadGateway, "502 Bad Gateway")
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Copy the response headers and status code
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.Error(moduleName, "Error reading payload response body:", err)
		replyHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error")
		return
	}
	_, err = w.Write(body)
	if err != nil {
		logging.Error(moduleName, "Error writing payload response body:", err)
	}
}

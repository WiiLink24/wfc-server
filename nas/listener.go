package nas

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type nasListener struct {
	net.Listener
}

type nasConn struct {
	net.Conn
	reader io.Reader
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

func listenAndServe(address string) {
	logging.Notice("NAS", "Starting HTTP server on", aurora.BrightCyan(address))

	l, err := net.Listen("tcp", address)
	common.ShouldNotError(err)
	listener := &nasListener{Listener: l}

	defer func() {
		common.ShouldNotError(listener.Close())
	}()

	err = server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

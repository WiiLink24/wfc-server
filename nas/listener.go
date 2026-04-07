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

	pr, pw := io.Pipe()
	go func() {
		r := bufio.NewReader(conn)
		filterDuplicateHost(r, pw)
		_, err := io.Copy(pw, r)
		_ = pw.CloseWithError(err)
	}()
	return &nasConn{
		Conn:   conn,
		reader: pr,
	}, nil
}

func (c *nasConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

// filterDuplicateHost wraps a net.Conn and filters out duplicate Host headers from the HTTP request,
// making the invalid requests sent by DWC acceptable to the standard library's HTTP server.
func filterDuplicateHost(r *bufio.Reader, p *io.PipeWriter) {
	// Read the first line of the HTTP request
	line, err := r.ReadString('\n')
	if err != nil {
		_, _ = p.Write([]byte(line))
		return
	}
	// Is this an HTTP request?
	if !strings.HasSuffix(line, "HTTP/1.1\r\n") {
		_, _ = p.Write([]byte(line))
		return
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

	_, _ = p.Write([]byte(line + headers.String()))
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

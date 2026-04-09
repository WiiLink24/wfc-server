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

type httpListener struct {
	net.Listener
}

type nasInConn struct {
	net.Conn
	in io.Reader
}

type nasIOConn struct {
	net.Conn
	in  io.Reader
	out io.Writer
}

func (c nasInConn) Read(b []byte) (int, error) { return c.in.Read(b) }

func (c nasInConn) Close() error {
	err := c.Conn.Close()
	if closer, ok := c.in.(io.Closer); ok {
		_ = closer.Close()
	}
	return err
}

func (c nasIOConn) Read(b []byte) (int, error) { return c.in.Read(b) }

func (c nasIOConn) Write(b []byte) (int, error) { return c.out.Write(b) }

func (c nasIOConn) Close() error {
	// Don't actually close the underlying connection here
	if closer, ok := c.in.(io.Closer); ok {
		_ = closer.Close()
	}
	if closer, ok := c.out.(io.Closer); ok {
		_ = closer.Close()
	}
	return nil
}

func (l *httpListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		r := bufio.NewReader(conn)
		if err := filterDuplicateHost(pw, r); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_, err := io.Copy(pw, r)
		_ = pw.CloseWithError(err)
	}()
	return &nasInConn{
		Conn: conn,
		in:   pr,
	}, nil
}

func listenAndServe() {
	logging.Notice("NAS", "Starting HTTP server on", aurora.BrightCyan(server.Addr))

	l, err := net.Listen("tcp", server.Addr)
	common.ShouldNotError(err)
	listener := &httpListener{Listener: l}

	defer func() {
		common.ShouldNotError(listener.Close())
	}()

	err = server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

type tlsListener struct {
	net.Listener
}

func (l *tlsListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// We're gonna need like 3 pipes here, from client -> tls -> filter -> server
	readFromServer, writeFromServer := io.Pipe()
	readToServer, writeToServer := io.Pipe()
	readToFilter, writeToFilter := io.Pipe()

	go func() {
		r := bufio.NewReader(readToFilter)
		if err := filterDuplicateHost(writeToServer, r); err != nil {
			_ = writeToServer.CloseWithError(err)
			return
		}
		_, err := io.Copy(writeToServer, r)
		_ = writeToServer.CloseWithError(err)
	}()
	go handleIncomingTLS(conn, readFromServer, writeToFilter)
	return &nasIOConn{
		Conn: conn,
		in:   readToServer,
		out:  writeFromServer,
	}, nil
}

func listenAndServeTLS() {
	logging.Notice("NAS", "Starting HTTPS server on", aurora.BrightCyan(tlsServer.Addr))

	l, err := net.Listen("tcp", tlsServer.Addr)
	common.ShouldNotError(err)
	listener := &tlsListener{Listener: l}

	defer func() {
		common.ShouldNotError(listener.Close())
	}()

	err = tlsServer.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

// filterDuplicateHost wraps a net.Conn and filters out duplicate Host headers from the HTTP request,
// making the invalid requests sent by DWC acceptable to the standard library's HTTP server.
func filterDuplicateHost(w *io.PipeWriter, r *bufio.Reader) error {
	// Read the first line of the HTTP request
	line, err := r.ReadString('\n')
	_, errPipe := w.Write([]byte(line))
	if err != nil || errPipe != nil {
		return errPipe
	}

	// Is this an HTTP request we handle?
	if !strings.HasSuffix(line, "HTTP/1.1\r\n") {
		return nil
	}

	// Iterate through the HTTP headers and remove any duplicate Host headers
	var headers bytes.Buffer
	hostSeen := false
	for {
		headerLine, err := r.ReadString('\n')
		_, wErr := w.Write([]byte(headerLine))
		if err != nil || wErr != nil || headerLine == "\r\n" || headerLine == "\n" {
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
	return errPipe
}

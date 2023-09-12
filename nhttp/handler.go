package nhttp

import (
	"context"
	"log"
	_http "net/http"
	"strings"
	"sync/atomic"
)

// serverHandler delegates to either the server's Handler or
// DefaultServeMux and also handles "OPTIONS *" requests.
type serverHandler struct {
	srv *Server
}

func (sh serverHandler) ServeHTTP(rw _http.ResponseWriter, req *_http.Request) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = _http.DefaultServeMux
	}

	if req.URL != nil && strings.Contains(req.URL.RawQuery, ";") {
		var allowQuerySemicolonsInUse int32
		req = req.WithContext(context.WithValue(req.Context(), &contextKey{"silence-semicolons"}, func() {
			atomic.StoreInt32(&allowQuerySemicolonsInUse, 1)
		}))
		defer func() {
			if atomic.LoadInt32(&allowQuerySemicolonsInUse) == 0 {
				log.Printf("nhttp: URL query contains semicolon, which is no longer a supported separator; parts of the query may be stripped when parsed; see golang.org/issue/25192\n")
			}
		}()
	}

	handler.ServeHTTP(rw, req)
}

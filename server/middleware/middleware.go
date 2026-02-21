package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
)

type Middleware struct {
	logger *slog.Logger
	debug  bool
	chain  http.Handler
}

func NewMiddleware(handler http.Handler, logger *slog.Logger, debug bool) *Middleware {
	m := &Middleware{
		logger: logger,
		debug:  debug,
	}
	m.chain = m.recover(m.log(handler))
	return m
}

func (m *Middleware) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.logger.Info("received request", "ip", r.RemoteAddr, "proto", r.Proto, "method", r.Method, "uri", r.URL.RequestURI())
		if m.debug {
			reqDump, _ := httputil.DumpRequest(r, true)
			readable := strings.ReplaceAll(string(reqDump), "\r\n", "\n")
			m.logger.Info(fmt.Sprintf("incoming request\n%s", readable))
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				m.logger.Error(fmt.Sprintf("%v", err), "method", r.Method, "uri", r.URL.RequestURI())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.chain.ServeHTTP(w, r)
}

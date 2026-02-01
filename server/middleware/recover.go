package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
)

type Recover struct {
	handler http.Handler
	logger  *slog.Logger
}

func NewRecoverMiddleware(handler http.Handler, logger *slog.Logger) *Recover {
	return &Recover{handler, logger}
}

func (rr *Recover) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.Header().Set("Connection", "close")
			rr.logger.Error(fmt.Sprintf("%v", err), "method", r.Method, "uri", r.URL.RequestURI())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}()
	rr.handler.ServeHTTP(w, r)
}

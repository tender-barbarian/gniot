package handlers

import (
	"log/slog"
	"net/http"
)


type Handlers struct {
	logger   *slog.Logger
}

func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers {
		logger:   logger,
	}
}

func (h *Handlers) WriteError(w http.ResponseWriter, r *http.Request, err error, msg string, statusCode int) {
	h.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
	http.Error(w, msg, statusCode)
}

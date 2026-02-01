package handlers

import (
	"log/slog"
	"net/http"
)

type ErrorHandler struct {
	logger *slog.Logger
}

func NewErrorHandler(logger *slog.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

func (e *ErrorHandler) WriteError(w http.ResponseWriter, r *http.Request, err error, msg string, statusCode int) {
	if err == nil {
		e.logger.Error(msg, "method", r.Method, "uri", r.URL.RequestURI())
	} else {
		e.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
	}

	http.Error(w, msg, statusCode)
}

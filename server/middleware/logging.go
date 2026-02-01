package middleware

import (
	"log/slog"
	"net/http"
)

type Logger struct {
	handler http.Handler
	logger  *slog.Logger
}

func NewLoggingMiddleware(handler http.Handler, logger *slog.Logger) *Logger {
	return &Logger{handler, logger}
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.logger.Info("received request", "ip", r.RemoteAddr, "proto", r.Proto, "method", r.Method, "uri", r.URL.RequestURI())
	l.handler.ServeHTTP(w, r)
}

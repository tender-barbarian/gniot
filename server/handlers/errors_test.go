package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewCustomHandlers(logger, nil, NewErrorHandler(logger))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	h.WriteError(rec, req, errors.New("internal"), "bad request", http.StatusBadRequest)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "bad request")
}

package handlers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tender-barbarian/gniot/repository/models"
	"github.com/tender-barbarian/gniot/service"
)

func TestExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewService[*models.Device, *models.Action, *models.Job](nil, nil, nil)
	errorHandler := NewErrorHandler(logger)
	h := NewCustomHandlers(errorHandler, svc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /execute", h.Execute)

	t.Run("nil body returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/execute", nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid JSON body")
	})

	t.Run("empty JSON returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/execute", strings.NewReader("{}"))
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid params")
	})

	t.Run("missing deviceId returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/execute", strings.NewReader(`{"actionId": 1}`))
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid params")
	})

	t.Run("missing actionId returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/execute", strings.NewReader(`{"deviceId": 1}`))
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid params")
	})

	t.Run("wrong type returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/execute", strings.NewReader(`{"deviceId": "abc"}`))
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid JSON body")
	})
}

package handlers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tender-barbarian/gniot/service"
)

func TestExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewService(nil, nil, nil)
	h := NewCustomHandlers(logger, svc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /execute", h.Execute)

	tests := []struct {
		name         string
		body         string
		wantCode     int
		wantContains string
	}{
		{
			name:         "nil body returns 400",
			body:         "",
			wantCode:     http.StatusBadRequest,
			wantContains: "invalid JSON body",
		},
		{
			name:         "empty JSON returns 400",
			body:         "{}",
			wantCode:     http.StatusBadRequest,
			wantContains: "invalid params",
		},
		{
			name:         "missing deviceId returns 400",
			body:         `{"actionId": 1}`,
			wantCode:     http.StatusBadRequest,
			wantContains: "invalid params",
		},
		{
			name:         "missing actionId returns 400",
			body:         `{"deviceId": 1}`,
			wantCode:     http.StatusBadRequest,
			wantContains: "invalid params",
		},
		{
			name:         "wrong type returns 400",
			body:         `{"deviceId": "abc"}`,
			wantCode:     http.StatusBadRequest,
			wantContains: "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest("POST", "/execute", nil)
			} else {
				req = httptest.NewRequest("POST", "/execute", strings.NewReader(tt.body))
			}
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantContains)
		})
	}
}

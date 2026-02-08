package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniot/service"
)

type mockExecutor struct {
	response *service.JSONRPCResponse
	err      error
}

func (m *mockExecutor) Execute(ctx context.Context, deviceId, actionId int) (*service.JSONRPCResponse, error) {
	return m.response, m.err
}

func TestExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("validation errors", func(t *testing.T) {
		h := NewCustomHandlers(logger, &mockExecutor{}, &ErrorHandler{logger: logger})
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
	})

	t.Run("service execution", func(t *testing.T) {
		tests := []struct {
			name           string
			mockResponse   *service.JSONRPCResponse
			mockErr        error
			wantCode       int
			wantContains   string
			validateResult func(t *testing.T, body string)
		}{
			{
				name: "successful execution returns device response",
				mockResponse: &service.JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  json.RawMessage(`{"ok":true}`),
					ID:      1,
				},
				wantCode: http.StatusOK,
				validateResult: func(t *testing.T, body string) {
					var resp service.JSONRPCResponse
					err := json.Unmarshal([]byte(body), &resp)
					require.NoError(t, err)
					assert.Equal(t, "2.0", resp.JSONRPC)
					assert.Equal(t, json.RawMessage(`{"ok":true}`), resp.Result)
				},
			},
			{
				name:         "service error returns 500",
				mockErr:      errors.New("device not found"),
				wantCode:     http.StatusInternalServerError,
				wantContains: "failed to execute",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := NewCustomHandlers(logger, &mockExecutor{response: tt.mockResponse, err: tt.mockErr}, &ErrorHandler{logger: logger})
				mux := http.NewServeMux()
				mux.HandleFunc("POST /execute", h.Execute)

				rec := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/execute", strings.NewReader(`{"deviceId":1,"actionId":1}`))
				mux.ServeHTTP(rec, req)

				assert.Equal(t, tt.wantCode, rec.Code)
				if tt.wantContains != "" {
					assert.Contains(t, rec.Body.String(), tt.wantContains)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, rec.Body.String())
				}
			})
		}
	})
}

package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	tests := []struct {
		name  string
		debug bool
		want  string
	}{
		{
			name:  "debug off - logs request, no dump",
			debug: false,
			want:  "received request",
		},
		{
			name:  "debug on - logs request and dump",
			debug: true,
			want:  "incoming request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			m := &Middleware{logger: slog.New(slog.NewTextHandler(&buf, nil)), debug: tc.debug}

			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("hello"))
			rec := httptest.NewRecorder()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("hello"))
			})
			m.log(next).ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "hello", rec.Body.String())
			assert.Contains(t, buf.String(), "received request")
			assert.Contains(t, buf.String(), tc.want)
		})
	}
}

func TestRecover(t *testing.T) {
	tests := []struct {
		name string
		next http.Handler
		want map[string]any
	}{
		{
			name: "no panic - passes through",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}),
			want: map[string]any{
				"code":       http.StatusOK,
				"body":       "ok",
				"connection": "",
				"log":        "",
			},
		},
		{
			name: "panic - returns 500 with Connection: close",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("something went wrong")
			}),
			want: map[string]any{
				"code":       http.StatusInternalServerError,
				"body":       "Internal Server Error\n",
				"connection": "close",
				"log":        "something went wrong",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			m := &Middleware{logger: slog.New(slog.NewTextHandler(&buf, nil))}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() {
				m.recover(tc.next).ServeHTTP(rec, req)
			})

			assert.Equal(t, tc.want["code"], rec.Code)
			assert.Equal(t, tc.want["body"], rec.Body.String())
			assert.Equal(t, tc.want["connection"], rec.Header().Get("Connection"))
			assert.Contains(t, buf.String(), tc.want["log"])
		})
	}
}

func TestServeHTTP_Chain(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
		want    map[string]any
	}{
		{
			name: "normal request passes through full chain",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}),
			want: map[string]any{
				"code":       http.StatusOK,
				"body":       "ok",
				"connection": "",
				"log":        "",
			},
		},
		{
			name: "panic in handler is recovered through chain",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("chain panic")
			}),
			want: map[string]any{
				"code":       http.StatusInternalServerError,
				"body":       "Internal Server Error\n",
				"connection": "close",
				"log":        "chain panic",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			m := NewMiddleware(tc.handler, slog.New(slog.NewTextHandler(&buf, nil)), false)

			req := httptest.NewRequest(http.MethodGet, "/chain", nil)
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() {
				m.ServeHTTP(rec, req)
			})

			assert.Equal(t, tc.want["code"], rec.Code)
			assert.Equal(t, tc.want["body"], rec.Body.String())
			assert.Equal(t, tc.want["connection"], rec.Header().Get("Connection"))
			assert.Contains(t, buf.String(), tc.want["log"])
		})
	}
}

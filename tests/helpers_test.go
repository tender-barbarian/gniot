package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniot/service"
)

type capturedRequest struct {
	mu          sync.Mutex
	Method      string
	Path        string
	ContentType string
	Body        service.JSONRPCRequest
}

func (c *capturedRequest) Get() capturedRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return capturedRequest{
		Method:      c.Method,
		Path:        c.Path,
		ContentType: c.ContentType,
		Body:        c.Body,
	}
}

func newMockDevice(t *testing.T, response string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.mu.Lock()
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&captured.Body) // nolint
		captured.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response)) // nolint
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

func createResource(t *testing.T, path, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		checkServerError(t, err)
	}
	defer resp.Body.Close() // nolint

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]int
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result["id"]
}

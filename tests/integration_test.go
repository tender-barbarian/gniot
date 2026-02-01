package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniot/repository/models"
	server "github.com/tender-barbarian/gniot/server"
)

const baseURL = "http://127.0.0.1:8080"

var serverErrCh chan error

func TestMain(m *testing.M) {
	serverErrCh = make(chan error, 1)
	go func() {
		if err := server.Run(); err != nil {
			serverErrCh <- err
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := waitForServer(ctx, baseURL)
	if err != nil {
		select {
		case serverErr := <-serverErrCh:
			fmt.Fprintf(os.Stderr, "server error: %v\n", serverErr)
		default:
			fmt.Fprintf(os.Stderr, "server not ready: %v\n", err)
		}
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func waitForServer(ctx context.Context, url string) error {
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close() // nolint
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func checkServerError(t *testing.T, err error) {
	t.Helper()
	select {
	case serverErr := <-serverErrCh:
		t.Fatalf("server error: %v (http error: %v)", serverErr, err)
	default:
		t.Fatalf("http error: %v", err)
	}
}

func TestDevicesRoutes_E2E(t *testing.T) {
	var createdID int

	t.Run("POST /devices creates device", func(t *testing.T) {
		body := `{"name":"esp32-sensor","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.100"}`
		resp, err := http.Post(baseURL+"/devices", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]int
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		createdID = result["id"]
		assert.Greater(t, createdID, 0)
	})

	t.Run("GET /devices/{id} returns device", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/devices/%d", baseURL, createdID))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var device models.Device
		if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		assert.Equal(t, createdID, device.ID)
		assert.Equal(t, "esp32-sensor", device.Name)
		assert.Equal(t, "192.168.1.100", device.IP)
	})

	t.Run("GET /devices returns list with device", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/devices")
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var devices []models.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		assert.GreaterOrEqual(t, len(devices), 1)
	})

	t.Run("POST /devices/{id} updates device", func(t *testing.T) {
		body := `{"name":"esp32-updated","type":"actuator","chip":"esp32","board":"devkit","ip":"192.168.1.101"}`
		resp, err := http.Post(fmt.Sprintf("%s/devices/%d", baseURL, createdID), "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		resp, err = http.Get(fmt.Sprintf("%s/devices/%d", baseURL, createdID))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		var device models.Device
		if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		assert.Equal(t, "esp32-updated", device.Name)
		assert.Equal(t, "192.168.1.101", device.IP)
	})

	t.Run("DELETE /devices/{id} removes device", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/devices/%d", baseURL, createdID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify deletion
		resp, err = http.Get(fmt.Sprintf("%s/devices/%d", baseURL, createdID))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestActionsRoutes_E2E(t *testing.T) {
	var createdID int

	t.Run("POST /actions creates action", func(t *testing.T) {
		body := `{"name":"toggle-led","path":"/api/led/toggle","params":"brightness=100"}`
		resp, err := http.Post(baseURL+"/actions", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]int
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		createdID = result["id"]
		assert.Greater(t, createdID, 0)
	})

	t.Run("GET /actions/{id} returns action", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/actions/%d", baseURL, createdID))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var action models.Action
		if err := json.NewDecoder(resp.Body).Decode(&action); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		assert.Equal(t, "toggle-led", action.Name)
		assert.Equal(t, "/api/led/toggle", action.Path)
	})

	t.Run("DELETE /actions/{id} removes action", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/actions/%d", baseURL, createdID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func createResource(t *testing.T, path, body string) int {
	t.Helper()
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() // nolint

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result["id"]
}

func TestExecuteRoute_E2E(t *testing.T) {
	// Start mock device server for success test
	mockDevice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockDevice.Close()

	// Setup
	actionID := createResource(t, "/actions", `{"name":"test-action","path":"toggle","params":"{}"}`)
	deviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"mock-device","type":"sensor","chip":"esp32","board":"devkit","ip":"%s","actions":"[%d]"}`, mockDevice.Listener.Addr().String(), actionID))
	unreachableDeviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"mock-unreachable-device","type":"sensor","chip":"esp32","board":"devkit","ip":"127.0.0.1:9999","actions":"[%d]"}`, actionID))

	t.Run("POST /execute with missing params returns 400", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBufferString(`{}`))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("POST /execute with non-existent device returns 500", func(t *testing.T) {
		body := `{"deviceId": 99999, "actionId": 1}`
		resp, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("POST /execute with action not belonging to device returns 500", func(t *testing.T) {
		// Create another action not assigned to device
		body := `{"name":"other-action","path":"other","params":"{}"}`
		resp, err := http.Post(baseURL+"/actions", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		var result map[string]int
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		otherActionID := result["id"]

		// Try to execute with action not assigned to device
		body = fmt.Sprintf(`{"deviceId": %d, "actionId": %d}`, deviceID, otherActionID)
		resp, err = http.Post(baseURL+"/execute", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("POST /execute succeeds", func(t *testing.T) {
		body := fmt.Sprintf(`{"deviceId": %d, "actionId": %d}`, deviceID, actionID)
		resp, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("POST /execute with unreachable device returns 500", func(t *testing.T) {
		body := fmt.Sprintf(`{"deviceId": %d, "actionId": %d}`, unreachableDeviceID, actionID)
		resp, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBufferString(body))
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		// Device is unreachable (127.0.0.1:9999), so should return 500
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

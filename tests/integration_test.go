package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func TestGenericRoutes_E2E(t *testing.T) {
	resources := []struct {
		name           string
		path           string
		createBody     string
		updateBody     string
		validateGet    func(t *testing.T, body []byte)
		validateUpdate func(t *testing.T, body []byte)
	}{
		{
			name:       "devices",
			path:       "/devices",
			createBody: `{"name":"esp32-sensor","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.100"}`,
			updateBody: `{"name":"esp32-updated","type":"actuator","chip":"esp32","board":"devkit","ip":"192.168.1.101"}`,
			validateGet: func(t *testing.T, body []byte) {
				var device models.Device
				require.NoError(t, json.Unmarshal(body, &device))
				assert.Equal(t, "esp32-sensor", device.Name)
				assert.Equal(t, "192.168.1.100", device.IP)
			},
			validateUpdate: func(t *testing.T, body []byte) {
				var device models.Device
				require.NoError(t, json.Unmarshal(body, &device))
				assert.Equal(t, "esp32-updated", device.Name)
				assert.Equal(t, "192.168.1.101", device.IP)
			},
		},
		{
			name:       "actions",
			path:       "/actions",
			createBody: `{"name":"toggle-led","path":"/api/led/toggle","params":"{\"brightness\":100}"}`,
			validateGet: func(t *testing.T, body []byte) {
				var action models.Action
				require.NoError(t, json.Unmarshal(body, &action))
				assert.Equal(t, "toggle-led", action.Name)
				assert.Equal(t, "/api/led/toggle", action.Path)
			},
			validateUpdate: func(t *testing.T, body []byte) {
				var action models.Action
				require.NoError(t, json.Unmarshal(body, &action))
				assert.Equal(t, "toggle-led-updated", action.Name)
				assert.Equal(t, "/api/led/toggle/updated", action.Path)
			},
		},
	}

	for _, res := range resources {
		t.Run(res.name, func(t *testing.T) {
			var createdID int

			t.Run("POST creates resource", func(t *testing.T) {
				resp, err := http.Post(baseURL+res.path, "application/json", bytes.NewBufferString(res.createBody))
				if err != nil {
					checkServerError(t, err)
				}
				defer resp.Body.Close() // nolint

				require.Equal(t, http.StatusCreated, resp.StatusCode)

				var result map[string]int
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				createdID = result["id"]
				assert.Greater(t, createdID, 0)
			})

			t.Run("GET returns resource", func(t *testing.T) {
				resp, err := http.Get(fmt.Sprintf("%s%s/%d", baseURL, res.path, createdID))
				if err != nil {
					checkServerError(t, err)
				}
				defer resp.Body.Close() // nolint

				require.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				res.validateGet(t, body)
			})

			t.Run("GET returns list", func(t *testing.T) {
				resp, err := http.Get(baseURL + res.path)
				if err != nil {
					checkServerError(t, err)
				}
				defer resp.Body.Close() // nolint

				require.Equal(t, http.StatusOK, resp.StatusCode)

				var items []json.RawMessage
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&items))
				assert.GreaterOrEqual(t, len(items), 1)
			})

			if res.updateBody != "" {
				t.Run("POST updates resource", func(t *testing.T) {
					resp, err := http.Post(fmt.Sprintf("%s%s/%d", baseURL, res.path, createdID), "application/json", bytes.NewBufferString(res.updateBody))
					if err != nil {
						checkServerError(t, err)
					}
					defer resp.Body.Close() // nolint

					require.Equal(t, http.StatusOK, resp.StatusCode)
				})

				t.Run("GET returns updated resource", func(t *testing.T) {
					resp, err := http.Get(fmt.Sprintf("%s%s/%d", baseURL, res.path, createdID))
					if err != nil {
						checkServerError(t, err)
					}
					defer resp.Body.Close() // nolint

					require.Equal(t, http.StatusOK, resp.StatusCode)

					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err)
					res.validateUpdate(t, body)
				})
			}

			t.Run("DELETE removes resource", func(t *testing.T) {
				req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s/%d", baseURL, res.path, createdID), nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					checkServerError(t, err)
				}
				defer resp.Body.Close() // nolint

				require.Equal(t, http.StatusOK, resp.StatusCode)
			})

			t.Run("GET returns 404 after deletion", func(t *testing.T) {
				resp, err := http.Get(fmt.Sprintf("%s%s/%d", baseURL, res.path, createdID))
				if err != nil {
					checkServerError(t, err)
				}
				defer resp.Body.Close() // nolint

				require.Equal(t, http.StatusNotFound, resp.StatusCode)
			})
		})
	}
}

func TestExecuteRoute_E2E(t *testing.T) {
	mockDevice, receivedReq := newMockDevice(t, `{"jsonrpc":"2.0","result":{"ok":true},"id":1}`)

	// Setup
	actionID := createResource(t, "/actions", `{"name":"test-action","path":"toggle","params":"{}"}`)
	deviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"mock-device","type":"sensor","chip":"esp32","board":"devkit","ip":"%s","actions":"[%d]"}`, mockDevice.Listener.Addr().String(), actionID))
	unreachableDeviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"mock-unreachable-device","type":"sensor","chip":"esp32","board":"devkit","ip":"127.0.0.1:9999","actions":"[%d]"}`, actionID))
	otherActionID := createResource(t, "/actions", `{"name":"other-action","path":"other","params":"{}"}`)

	tests := []struct {
		name        string
		body        string
		wantCode    int
		validateReq func(t *testing.T)
	}{
		{
			name:     "missing params returns 400",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "non-existent device returns 500",
			body:     `{"deviceId": 99999, "actionId": 1}`,
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "action not belonging to device returns 500",
			body:     fmt.Sprintf(`{"deviceId": %d, "actionId": %d}`, deviceID, otherActionID),
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "successful execution",
			body:     fmt.Sprintf(`{"deviceId": %d, "actionId": %d}`, deviceID, actionID),
			wantCode: http.StatusOK,
			validateReq: func(t *testing.T) {
				assert.Equal(t, http.MethodPost, receivedReq.Method)
				assert.Equal(t, "/rpc", receivedReq.Path)
				assert.Equal(t, "application/json", receivedReq.ContentType)
				assert.Equal(t, "2.0", receivedReq.Body.JSONRPC)
				assert.Equal(t, "toggle", receivedReq.Body.Method)
				assert.Equal(t, 1, receivedReq.Body.ID)
			},
		},
		{
			name:     "unreachable device returns 500",
			body:     fmt.Sprintf(`{"deviceId": %d, "actionId": %d}`, unreachableDeviceID, actionID),
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBufferString(tt.body))
			if err != nil {
				checkServerError(t, err)
			}
			defer resp.Body.Close() // nolint

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			if tt.validateReq != nil {
				tt.validateReq(t)
			}
		})
	}
}

func TestDeviceActionValidation_E2E(t *testing.T) {
	// Create a valid action first
	validActionID := createResource(t, "/actions", `{"name":"valid-action","path":"test","params":"{}"}`)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "create device with non-existent action fails",
			body:     `{"name":"test-device1","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.50","actions":"[99999]"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create device with mixed valid and invalid actions fails",
			body:     fmt.Sprintf(`{"name":"test-device2","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.51","actions":"[%d, 99999]"}`, validActionID),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create device with valid action succeeds",
			body:     fmt.Sprintf(`{"name":"test-device3","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.52","actions":"[%d]"}`, validActionID),
			wantCode: http.StatusCreated,
		},
		{
			name:     "create device with empty actions succeeds",
			body:     `{"name":"test-device4","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.53","actions":""}`,
			wantCode: http.StatusCreated,
		},
		{
			name:     "create device without actions field succeeds",
			body:     `{"name":"test-device5","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.54"}`,
			wantCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(baseURL+"/devices", "application/json", bytes.NewBufferString(tt.body))
			if err != nil {
				checkServerError(t, err)
			}
			defer resp.Body.Close() // nolint

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}

	t.Run("update device with non-existent action fails", func(t *testing.T) {
		// Create a device without actions
		deviceID := createResource(t, "/devices", `{"name":"update-test","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.60"}`)

		// Try to update with non-existent action
		resp, err := http.Post(
			fmt.Sprintf("%s/devices/%d", baseURL, deviceID),
			"application/json",
			bytes.NewBufferString(`{"actions":"[99999]"}`),
		)
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("update device with valid action succeeds", func(t *testing.T) {
		// Create a device without actions
		deviceID := createResource(t, "/devices", `{"name":"update-test-2","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.61"}`)

		// Update with valid action
		resp, err := http.Post(
			fmt.Sprintf("%s/devices/%d", baseURL, deviceID),
			"application/json",
			bytes.NewBufferString(fmt.Sprintf(`{"actions":"[%d]"}`, validActionID)),
		)
		if err != nil {
			checkServerError(t, err)
		}
		defer resp.Body.Close() // nolint

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

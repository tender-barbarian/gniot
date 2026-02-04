package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniot/repository/models"
	server "github.com/tender-barbarian/gniot/server"
	"github.com/tender-barbarian/gniot/service"
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
	futureTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

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
		{
			name:       "jobs",
			path:       "/jobs",
			createBody: fmt.Sprintf(`{"name":"test-job","devices":"[1]","action":"1","run_at":"%s","interval":"1h","enabled":1}`, futureTime),
			updateBody: `{"name":"test-job-updated","interval":"2h","enabled":0}`,
			validateGet: func(t *testing.T, body []byte) {
				var job models.Job
				require.NoError(t, json.Unmarshal(body, &job))
				assert.Equal(t, "test-job", job.Name)
				assert.Equal(t, "1h", job.Interval)
				assert.Equal(t, 1, job.Enabled)
			},
			validateUpdate: func(t *testing.T, body []byte) {
				var job models.Job
				require.NoError(t, json.Unmarshal(body, &job))
				assert.Equal(t, "test-job-updated", job.Name)
				assert.Equal(t, "2h", job.Interval)
				assert.Equal(t, 0, job.Enabled)
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
	// Start mock device server for success test
	mockDevice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockDevice.Close()

	// Setup
	actionID := createResource(t, "/actions", `{"name":"test-action","path":"toggle","params":"{}"}`)
	deviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"mock-device","type":"sensor","chip":"esp32","board":"devkit","ip":"%s","actions":"[%d]"}`, mockDevice.Listener.Addr().String(), actionID))
	unreachableDeviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"mock-unreachable-device","type":"sensor","chip":"esp32","board":"devkit","ip":"127.0.0.1:9999","actions":"[%d]"}`, actionID))
	otherActionID := createResource(t, "/actions", `{"name":"other-action","path":"other","params":"{}"}`)

	tests := []struct {
		name     string
		body     string
		wantCode int
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
		})
	}
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

func TestJobRunner_E2E(t *testing.T) {
	// Start mock device server
	mockDevice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req service.JSONRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			t.Fatal(err)
			return
		}

		assert.Equal(t, "toggle", req.Method)
		assert.Equal(t, map[string]any{}, req.Params)

		w.WriteHeader(http.StatusOK)
	}))
	defer mockDevice.Close()

	// Setup action and device
	actionID := createResource(t, "/actions", `{"name":"job-action","path":"toggle","params":"{}"}`)
	deviceID := createResource(t, "/devices", fmt.Sprintf(`{"name":"job-device","type":"sensor","chip":"esp32","board":"devkit","ip":"%s","actions":"[%d]"}`, mockDevice.Listener.Addr().String(), actionID))

	// Create job with time in the past (should execute on next tick)
	pastTime := time.Now().Add(-1 * time.Second).Format(time.RFC3339)
	jobBody := fmt.Sprintf(`{"name":"immediate-job","devices":"[%d]","action":"%d","run_at":"%s","interval":"1h","enabled":1}`, deviceID, actionID, pastTime)
	jobID := createResource(t, "/jobs", jobBody)

	t.Run("job is executed", func(t *testing.T) {
		time.Sleep(10 * time.Second)
		job := getResource[models.Job](t, fmt.Sprintf("/jobs/%d", jobID))
		updatedTime, _ := time.Parse(time.RFC3339, job.RunAt)
		assert.True(t, updatedTime.After(time.Now().Add(59*time.Minute))) // Should be ~1h from now
		assert.NotEmpty(t, job.LastTriggered)
		assert.NotEmpty(t, job.LastCheck)
	})
}

func getResource[T any](t *testing.T, path string) T {
	t.Helper()
	resp, err := http.Get(baseURL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() // nolint

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
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
			body:     `{"name":"test-device","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.50","actions":"[99999]"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create device with mixed valid and invalid actions fails",
			body:     fmt.Sprintf(`{"name":"test-device","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.51","actions":"[%d, 99999]"}`, validActionID),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create device with valid action succeeds",
			body:     fmt.Sprintf(`{"name":"test-device","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.52","actions":"[%d]"}`, validActionID),
			wantCode: http.StatusCreated,
		},
		{
			name:     "create device with empty actions succeeds",
			body:     `{"name":"test-device","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.53","actions":""}`,
			wantCode: http.StatusCreated,
		},
		{
			name:     "create device without actions field succeeds",
			body:     `{"name":"test-device","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.54"}`,
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

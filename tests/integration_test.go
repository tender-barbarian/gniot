package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniotek/repository/models"
	server "github.com/tender-barbarian/gniotek/server"
	"gopkg.in/yaml.v3"
)

const baseURL = "http://127.0.0.1:8080"

var serverErrCh chan error

func TestMain(m *testing.M) {
	os.Setenv("AUTOMATIONS_INTERVAL", "1s") // nolint

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

func TestDevices_CRUD(t *testing.T) {
	expectedResources := []models.Device{
		{Name: "esp32-sensor", Type: "sensor", Chip: "esp32", Board: "devkit", IP: "192.168.1.100"},
		{Name: "esp32-relay", Type: "actuator", Chip: "esp32", Board: "devkit", IP: "192.168.1.101"},
		{Name: "esp8266-temp", Type: "sensor", Chip: "esp8266", Board: "nodemcu", IP: "192.168.1.102"},
	}

	ids := make([]int, len(expectedResources))
	for i, expectedResource := range expectedResources {
		body, err := json.Marshal(expectedResource)
		require.NoError(t, err)
		ids[i] = createResource(t, "/devices", string(body))
	}

	t.Run("test device get", func(t *testing.T) {
		actualResource := getResource[models.Device](t, "/devices", ids[0])
		assert.Equal(t, expectedResources[0].Name, actualResource.Name)
		assert.Equal(t, expectedResources[0].IP, actualResource.IP)
	})

	t.Run("test device getAll", func(t *testing.T) {
		actualResources := getAllResources[models.Device](t, "/devices")
		require.Equal(t, len(expectedResources), len(actualResources))
		for i, expectedResource := range expectedResources {
			assert.Equal(t, expectedResource.Name, actualResources[i].Name, "device[%d] Name", i)
			assert.Equal(t, expectedResource.Type, actualResources[i].Type, "device[%d] Type", i)
			assert.Equal(t, expectedResource.Chip, actualResources[i].Chip, "device[%d] Chip", i)
			assert.Equal(t, expectedResource.Board, actualResources[i].Board, "device[%d] Board", i)
			assert.Equal(t, expectedResource.IP, actualResources[i].IP, "device[%d] IP", i)
		}
	})

	t.Run("test device update", func(t *testing.T) {
		wantUpdated := models.Device{Name: "esp32-updated", Type: "actuator", Chip: "esp32", Board: "devkit", IP: "192.168.1.200"}
		updateResource(t, "/devices", ids[0], fmt.Sprintf(
			`{"name":"%s","type":"%s","chip":"%s","board":"%s","ip":"%s"}`,
			wantUpdated.Name, wantUpdated.Type, wantUpdated.Chip, wantUpdated.Board, wantUpdated.IP,
		))

		updated := getResource[models.Device](t, "/devices", ids[0])
		assert.Equal(t, wantUpdated.Name, updated.Name)
		assert.Equal(t, wantUpdated.IP, updated.IP)
	})

	t.Run("test device delete", func(t *testing.T) {
		for _, id := range ids {
			deleteResource(t, "/devices", id)
			assertNotFound(t, "/devices", id)
		}
	})
}

func TestDevices_ActionValidation(t *testing.T) {
	validActionID := createResource(t, "/actions", `{"name":"valid-action","path":"test","params":"{}"}`)
	deviceID := createResource(t, "/devices", `{"name":"dev-for-update","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.60"}`)

	tests := []struct {
		name     string
		path     string
		body     string
		wantCode int
	}{
		{
			name:     "create with non-existent action fails",
			path:     "/devices",
			body:     `{"name":"d1","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.50","actions":"[99999]"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create with mixed valid and invalid actions fails",
			path:     "/devices",
			body:     fmt.Sprintf(`{"name":"d2","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.51","actions":"[%d, 99999]"}`, validActionID),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create with valid action succeeds",
			path:     "/devices",
			body:     fmt.Sprintf(`{"name":"d3","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.52","actions":"[%d]"}`, validActionID),
			wantCode: http.StatusCreated,
		},
		{
			name:     "create with empty actions succeeds",
			path:     "/devices",
			body:     `{"name":"d4","type":"sensor","chip":"esp32","board":"devkit","ip":"192.168.1.53","actions":""}`,
			wantCode: http.StatusCreated,
		},
		{
			name:     "update with non-existent action fails",
			path:     fmt.Sprintf("/devices/%d", deviceID),
			body:     `{"actions":"[99999]"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "update with valid action succeeds",
			path:     fmt.Sprintf("/devices/%d", deviceID),
			body:     fmt.Sprintf(`{"actions":"[%d]"}`, validActionID),
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(baseURL+tt.path, "application/json", bytes.NewBufferString(tt.body))
			if err != nil {
				checkServerError(t, err)
			}
			defer resp.Body.Close() // nolint

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}

	t.Cleanup(func() {
		deleteResource(t, "/actions", validActionID)
		deleteResource(t, "/devices", deviceID)
	})
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

func TestAutomations_DefinitionValidation(t *testing.T) {
	readTempID := createResource(t, "/actions", `{"name":"read-temp","path":"read_temp","params":"{}"}`)
	turnOnID := createResource(t, "/actions", `{"name":"turn-on","path":"turn_on","params":"{}"}`)
	unassignedActionID := createResource(t, "/actions", `{"name":"unassigned-action","path":"unassigned","params":"{}"}`)

	triggerDevice, triggerReq := newMockDevice(t, `{"jsonrpc":"2.0","result":{"temperature":30},"id":1}`)
	triggerDeviceID := createResource(t, "/devices", fmt.Sprintf(
		`{"name":"sensor-1","type":"sensor","chip":"esp32","board":"devkit","ip":"%s","actions":"[%d]"}`, triggerDevice.Listener.Addr().String(), readTempID))

	actionDevice, actionReq := newMockDevice(t, `{"jsonrpc":"2.0","result":{"ok":true},"id":1}`)
	actionDeviceID := createResource(t, "/devices", fmt.Sprintf(
		`{"name":"actuator-1","type":"actuator","chip":"esp32","board":"devkit","ip":"%s","actions":"[%d]"}`, actionDevice.Listener.Addr().String(), turnOnID))

	tests := []struct {
		name               string
		interval           string
		triggers           []models.AutomationTrigger
		actions            []models.AutomationAction
		wantCode           int
		validateTriggerReq func(t *testing.T)
		validateActionReq  func(t *testing.T)
	}{
		{
			name:     "trigger referencing non-existent device",
			interval: "5m",
			triggers: []models.AutomationTrigger{{Device: "no-such-device", Action: "read-temp", Conditions: []models.AutomationCondition{
				{Field: "temperature", Operator: ">", Threshold: 25},
			}}},
			actions:  []models.AutomationAction{{Device: "actuator-1", Action: "turn-on"}},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "trigger referencing non-existent action",
			interval: "5m",
			triggers: []models.AutomationTrigger{{Device: "sensor-1", Action: "no-such-action", Conditions: []models.AutomationCondition{
				{Field: "temperature", Operator: ">", Threshold: 25},
			}}},
			actions:  []models.AutomationAction{{Device: "actuator-1", Action: "turn-on"}},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "trigger action not assigned to device",
			interval: "5m",
			triggers: []models.AutomationTrigger{{Device: "sensor-1", Action: "turn-on", Conditions: []models.AutomationCondition{
				{Field: "temperature", Operator: ">", Threshold: 25},
			}}},
			actions:  []models.AutomationAction{{Device: "actuator-1", Action: "turn-on"}},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "action device not found",
			interval: "5m",
			triggers: []models.AutomationTrigger{{Device: "sensor-1", Action: "read-temp", Conditions: []models.AutomationCondition{
				{Field: "temperature", Operator: ">", Threshold: 25},
			}}},
			actions:  []models.AutomationAction{{Device: "no-such-device", Action: "turn-on"}},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "action not assigned to device",
			interval: "5m",
			triggers: []models.AutomationTrigger{{Device: "sensor-1", Action: "read-temp", Conditions: []models.AutomationCondition{
				{Field: "temperature", Operator: ">", Threshold: 25},
			}}},
			actions:  []models.AutomationAction{{Device: "actuator-1", Action: "unassigned-action"}},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "valid definition succeeds",
			interval: "1s",
			triggers: []models.AutomationTrigger{{Device: "sensor-1", Action: "read-temp", Conditions: []models.AutomationCondition{
				{Field: "temperature", Operator: ">", Threshold: 25},
			}}},
			actions:  []models.AutomationAction{{Device: "actuator-1", Action: "turn-on"}},
			wantCode: http.StatusCreated,
			validateTriggerReq: func(t *testing.T) {
				snap := triggerReq.Get()
				assert.Equal(t, http.MethodPost, snap.Method)
				assert.Equal(t, "/rpc", snap.Path)
				assert.Equal(t, "application/json", snap.ContentType)
				assert.Equal(t, "2.0", snap.Body.JSONRPC)
				assert.Equal(t, "read_temp", snap.Body.Method)
				assert.Equal(t, 1, snap.Body.ID)
			},
			validateActionReq: func(t *testing.T) {
				snap := actionReq.Get()
				assert.Equal(t, http.MethodPost, snap.Method)
				assert.Equal(t, "/rpc", snap.Path)
				assert.Equal(t, "application/json", snap.ContentType)
				assert.Equal(t, "2.0", snap.Body.JSONRPC)
				assert.Equal(t, "turn_on", snap.Body.Method)
				assert.Equal(t, 1, snap.Body.ID)
			},
		},
	}

	var createdAutomationIDs []int

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := models.AutomationDefinition{
				Interval: tt.interval,
				Triggers: tt.triggers,
				Actions:  tt.actions,
			}
			defYAML, err := yaml.Marshal(def)
			require.NoError(t, err)

			body, err := json.Marshal(map[string]any{
				"name":       tt.name,
				"enabled":    true,
				"definition": string(defYAML),
			})
			require.NoError(t, err)

			resp, err := http.Post(baseURL+"/automations", "application/json", bytes.NewBuffer(body))
			if err != nil {
				checkServerError(t, err)
			}
			defer resp.Body.Close() // nolint

			assert.Equal(t, tt.wantCode, resp.StatusCode)

			if resp.StatusCode == http.StatusCreated {
				var result map[string]int
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				createdAutomationIDs = append(createdAutomationIDs, result["id"])
			}

			if tt.validateTriggerReq != nil {
				require.Eventually(t, func() bool {
					return triggerReq.Get().Method != ""
				}, 10*time.Second, 200*time.Millisecond, "automation should have called a trigger device")
				tt.validateTriggerReq(t)
			}

			if tt.validateActionReq != nil {
				require.Eventually(t, func() bool {
					return actionReq.Get().Method != ""
				}, 10*time.Second, 200*time.Millisecond, "automation should have called an action device")
				tt.validateActionReq(t)
			}
		})
	}

	t.Cleanup(func() {
		for _, id := range createdAutomationIDs {
			deleteResource(t, "/automations", id)
		}
		deleteResource(t, "/devices", triggerDeviceID)
		deleteResource(t, "/devices", actionDeviceID)
		deleteResource(t, "/actions", readTempID)
		deleteResource(t, "/actions", turnOnID)
		deleteResource(t, "/actions", unassignedActionID)
	})
}

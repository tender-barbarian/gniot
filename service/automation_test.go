package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniot/cache"
	"github.com/tender-barbarian/gniot/repository/models"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Test Helper Functions
// ============================================================================

// createTestServiceForAutomation creates a Service instance with real caches and mock repos
// Uses the shared mocks from mocks_test.go
func createTestServiceForAutomation(
	deviceRepo *mockDeviceRepo,
	actionRepo *mockActionRepo,
	automationRepo *mockAutomationRepo,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// Build mockQuerier from device/action repo data
	nameToID := make(map[string]int)
	if deviceRepo != nil {
		for _, d := range deviceRepo.devices {
			nameToID["devices:"+d.Name] = d.ID
		}
	}
	if actionRepo != nil {
		for _, a := range actionRepo.actions {
			nameToID["actions:"+a.Name] = a.ID
		}
	}
	querier := &mockQuerier{nameToID: nameToID}

	return NewService(ServiceConfig{
		DevicesRepo:     deviceRepo,
		ActionsRepo:     actionRepo,
		AutomationsRepo: automationRepo,
		QueryRepo:       querier,
		DevicesCache:    cache.NewCache[*models.Device](),
		ActionsCache:    cache.NewCache[*models.Action](),
		Logger:          logger,
	})
}

// createYAMLDefinition creates a YAML string from an AutomationDefinition
func createYAMLDefinition(def models.AutomationDefinition) (string, error) {
	data, err := yaml.Marshal(&def)
	if err != nil {
		return "", fmt.Errorf("failed to marshal automation definition: %w", err)
	}
	return string(data), nil
}

// createPastTimestamp creates an RFC3339 timestamp in the past
func createPastTimestamp(ago time.Duration) string {
	return time.Now().Add(-ago).Format(time.RFC3339)
}

// ============================================================================
// TESTS: Orchestration
// ============================================================================

// TestProcessAutomations tests processAutomations orchestration logic.
func TestProcessAutomations(t *testing.T) {
	ctx := context.Background()

	t.Run("conditions met - action executes", func(t *testing.T) {
		server := createRecordingServer(`{"jsonrpc":"2.0","result":{"temperature":30.0},"id":1}`, http.StatusOK)
		defer server.Close()

		yamlDef, err := createYAMLDefinition(models.AutomationDefinition{
			Interval: "5m",
			Triggers: []models.AutomationTrigger{
				{Device: "sensor1", Action: "read_temp", Conditions: []models.AutomationCondition{
					{Field: "temperature", Operator: ">", Threshold: 25.0},
				}},
			},
			Actions: []models.AutomationAction{
				{Device: "heater", Action: "turn_off"},
			},
		})
		require.NoError(t, err)

		automation := &models.Automation{
			ID:              1,
			Name:            "temp_control",
			Enabled:         true,
			Definition:      yamlDef,
			LastTriggersRun: createPastTimestamp(10 * time.Minute),
		}

		automationRepo := &mockAutomationRepo{automations: []*models.Automation{automation}}

		svc := createTestServiceForAutomation(
			&mockDeviceRepo{
				devices: []*models.Device{
					{ID: 1, Name: "sensor1", IP: server.Listener.Addr().String(), Actions: "[1]"},
					{ID: 2, Name: "heater", IP: server.Listener.Addr().String(), Actions: "[2]"},
				},
			},
			&mockActionRepo{
				actions: []*models.Action{
					{ID: 1, Name: "read_temp", Path: "read_temp", Params: `{}`},
					{ID: 2, Name: "turn_off", Path: "turn_off", Params: `{}`},
				},
			},
			automationRepo,
			nil,
		)

		err = svc.processAutomations(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, server.getCallCount()) // trigger + action

		// Validate actual executions
		requests := server.getRequests()
		assert.Len(t, requests, 2)
		assert.Equal(t, "read_temp", requests[0].Method)
		assert.Equal(t, "turn_off", requests[1].Method)

		updated := automationRepo.getUpdated()
		assert.NotNil(t, updated)
		parsedTime, _ := time.Parse(time.RFC3339, updated.LastCheck)
		assert.WithinDuration(t, time.Now(), parsedTime, time.Second)
	})

	t.Run("conditions not met - action skipped", func(t *testing.T) {
		server := createRecordingServer(`{"jsonrpc":"2.0","result":{"temperature":20.0},"id":1}`, http.StatusOK)
		defer server.Close()

		yamlDef, err := createYAMLDefinition(models.AutomationDefinition{
			Interval: "5m",
			Triggers: []models.AutomationTrigger{
				{Device: "sensor1", Action: "read_temp", Conditions: []models.AutomationCondition{
					{Field: "temperature", Operator: ">", Threshold: 25.0},
				}},
			},
			Actions: []models.AutomationAction{
				{Device: "heater", Action: "turn_off"},
			},
		})
		require.NoError(t, err)

		automation := &models.Automation{
			ID:              1,
			Name:            "temp_control",
			Enabled:         true,
			Definition:      yamlDef,
			LastTriggersRun: createPastTimestamp(10 * time.Minute),
		}

		svc := createTestServiceForAutomation(
			&mockDeviceRepo{
				devices: []*models.Device{{ID: 1, Name: "sensor1", IP: server.Listener.Addr().String(), Actions: "[1]"}},
			},
			&mockActionRepo{
				actions: []*models.Action{{ID: 1, Name: "read_temp", Path: "read_temp", Params: `{}`}},
			},
			&mockAutomationRepo{automations: []*models.Automation{automation}},
			nil,
		)

		err = svc.processAutomations(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, server.getCallCount()) // only trigger, no action

		// Validate only trigger was executed
		requests := server.getRequests()
		assert.Len(t, requests, 1)
		assert.Equal(t, "read_temp", requests[0].Method)
	})

	t.Run("error getting automations", func(t *testing.T) {
		svc := createTestServiceForAutomation(nil, nil, &mockAutomationRepo{err: errors.New("db error")}, nil)
		err := svc.processAutomations(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "getting automations")
	})
}

// ============================================================================
// TESTS: Pure Functions
// ============================================================================

// TestEvaluateOperator tests the evaluateOperator function with all operators
// and edge cases. This is a pure function with no dependencies.
func TestEvaluateOperator(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		operator  string
		threshold float64
		want      bool
	}{
		// All six operators
		{name: "> operator", value: 10.5, operator: ">", threshold: 5.0, want: true},
		{name: "< operator", value: 3.0, operator: "<", threshold: 5.0, want: true},
		{name: ">= operator when equal", value: 5.0, operator: ">=", threshold: 5.0, want: true},
		{name: "<= operator when equal", value: 5.0, operator: "<=", threshold: 5.0, want: true},
		{name: "== operator", value: 5.0, operator: "==", threshold: 5.0, want: true},
		{name: "!= operator", value: 10.0, operator: "!=", threshold: 5.0, want: true},

		// Edge cases
		{name: "negative numbers", value: -3.0, operator: ">", threshold: -5.0, want: true},
		{name: "zero comparison", value: 0.0, operator: "==", threshold: 0.0, want: true},
		{name: "invalid operator", value: 10.0, operator: "~", threshold: 5.0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateOperator(tt.value, tt.operator, tt.threshold)
			assert.Equal(t, tt.want, got, "evaluateOperator(%v, %q, %v) = %v, want %v",
				tt.value, tt.operator, tt.threshold, got, tt.want)
		})
	}
}

// TestGetFieldValue tests the getFieldValue function for extracting values
// from nested JSON objects. Tests simple fields, nested paths, and error cases.
func TestGetFieldValue(t *testing.T) {
	// Create a service instance (needed since getFieldValue is a method)
	svc := createTestServiceForAutomation(nil, nil, nil, nil)

	tests := []struct {
		name      string
		data      map[string]any
		field     string
		want      float64
		wantError bool
		errorMsg  string
	}{
		// Simple field extraction - various numeric types
		{
			name:  "simple float64 field",
			data:  map[string]any{"temperature": 25.5},
			field: "temperature",
			want:  25.5,
		},
		{
			name:  "simple int field",
			data:  map[string]any{"count": 42},
			field: "count",
			want:  42.0,
		},
		{
			name:  "simple int64 field",
			data:  map[string]any{"timestamp": int64(1234567890)},
			field: "timestamp",
			want:  1234567890.0,
		},
		{
			name:  "negative and zero values",
			data:  map[string]any{"value": -10.5},
			field: "value",
			want:  -10.5,
		},

		// Nested field extraction
		{
			name: "nested one level",
			data: map[string]any{
				"sensor": map[string]any{"temperature": 30.0},
			},
			field: "sensor.temperature",
			want:  30.0,
		},
		{
			name: "deeply nested fields",
			data: map[string]any{
				"device": map[string]any{
					"sensor": map[string]any{
						"reading": map[string]any{"value": 45.2},
					},
				},
			},
			field: "device.sensor.reading.value",
			want:  45.2,
		},

		// Real-world JSON structure
		{
			name: "JSON-RPC result structure",
			data: map[string]any{
				"result": map[string]any{
					"temperature": 22.5,
					"humidity":    65.0,
				},
			},
			field: "result.temperature",
			want:  22.5,
		},

		// Error cases - field not found
		{
			name:      "field not found",
			data:      map[string]any{"temperature": 25.0},
			field:     "humidity",
			wantError: true,
			errorMsg:  "field 'humidity' not found",
		},
		{
			name: "nested field not found",
			data: map[string]any{
				"sensor": map[string]any{"temperature": 25.0},
			},
			field:     "sensor.humidity",
			wantError: true,
			errorMsg:  "field 'humidity' not found",
		},

		// Error cases - field is not an object
		{
			name:      "navigating through non-object",
			data:      map[string]any{"temperature": 25.0},
			field:     "temperature.value",
			wantError: true,
			errorMsg:  "field 'value' is not an object",
		},

		// Error cases - field is not a number
		{
			name:      "field is string",
			data:      map[string]any{"name": "sensor1"},
			field:     "name",
			wantError: true,
			errorMsg:  "field 'name' is not a number",
		},
		{
			name: "field is object",
			data: map[string]any{
				"sensor": map[string]any{"temperature": 25.0},
			},
			field:     "sensor",
			wantError: true,
			errorMsg:  "field 'sensor' is not a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.getFieldValue(tt.data, tt.field)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestApplyConditionLogic tests the applyConditionLogic function for combining
// boolean condition results with AND/OR logic.
func TestApplyConditionLogic(t *testing.T) {
	// Create a service instance (needed since applyConditionLogic is a method)
	svc := createTestServiceForAutomation(nil, nil, nil, nil)

	tests := []struct {
		name    string
		results []bool
		logic   string
		want    bool
	}{
		// AND logic
		{name: "AND - all true", results: []bool{true, true, true}, logic: "and", want: true},
		{name: "AND - one false", results: []bool{true, false, true}, logic: "and", want: false},
		{name: "AND - single true", results: []bool{true}, logic: "and", want: true},

		// OR logic
		{name: "OR - all false", results: []bool{false, false, false}, logic: "or", want: false},
		{name: "OR - one true", results: []bool{false, true, false}, logic: "or", want: true},
		{name: "OR - all true", results: []bool{true, true}, logic: "or", want: true},

		// Default/edge cases
		{name: "empty logic defaults to AND", results: []bool{true, false}, logic: "", want: false},
		{name: "invalid logic defaults to AND", results: []bool{true, true}, logic: "invalid", want: true},
		{name: "empty results returns true", results: []bool{}, logic: "and", want: true},
		{name: "nil results returns true", results: nil, logic: "and", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.applyConditionLogic(tt.results, tt.logic)
			assert.Equal(t, tt.want, got, "applyConditionLogic(%v, %q) = %v, want %v",
				tt.results, tt.logic, got, tt.want)
		})
	}
}

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/tender-barbarian/gniot/repository/models"
)

// ============================================================================
// Mock Device Repository
// ============================================================================

type mockDeviceRepo struct {
	device  *models.Device
	devices []*models.Device
	err     error
}

func (m *mockDeviceRepo) Create(ctx context.Context, model *models.Device) (int, error) {
	return 0, nil
}

func (m *mockDeviceRepo) Get(ctx context.Context, id int) (*models.Device, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Try to find device by ID in devices slice
	if m.devices != nil {
		for _, device := range m.devices {
			if device.ID == id {
				return device, nil
			}
		}
	}
	// Fall back to single device field
	return m.device, nil
}

func (m *mockDeviceRepo) GetAll(ctx context.Context) ([]*models.Device, error) {
	if m.devices != nil {
		return m.devices, m.err
	}
	return nil, nil
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id int) error {
	return nil
}

func (m *mockDeviceRepo) Update(ctx context.Context, model *models.Device, id int) error {
	return nil
}

func (m *mockDeviceRepo) GetTable() string {
	return "devices"
}

func (m *mockDeviceRepo) GetDB() *sql.DB {
	return nil
}

// ============================================================================
// Mock Action Repository
// ============================================================================

type mockActionRepo struct {
	action  *models.Action
	actions []*models.Action
	err     error
}

func (m *mockActionRepo) Create(ctx context.Context, model *models.Action) (int, error) {
	return 0, nil
}

func (m *mockActionRepo) Get(ctx context.Context, id int) (*models.Action, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Try to find action by ID in actions slice
	if m.actions != nil {
		for _, action := range m.actions {
			if action.ID == id {
				return action, nil
			}
		}
	}
	// Fall back to single action field
	return m.action, nil
}

func (m *mockActionRepo) GetAll(ctx context.Context) ([]*models.Action, error) {
	if m.actions != nil {
		return m.actions, m.err
	}
	return nil, nil
}

func (m *mockActionRepo) Delete(ctx context.Context, id int) error {
	return nil
}

func (m *mockActionRepo) Update(ctx context.Context, model *models.Action, id int) error {
	return nil
}

func (m *mockActionRepo) GetTable() string {
	return "actions"
}

func (m *mockActionRepo) GetDB() *sql.DB {
	return nil
}

// ============================================================================
// Recording Server (for verification)
// ============================================================================

type recordingServer struct {
	*httptest.Server
	callCount int
	requests  []JSONRPCRequest
	response  string
	mu        sync.Mutex
}

func createRecordingServer(response string, statusCode int) *recordingServer {
	rs := &recordingServer{
		response:  response,
		requests:  make([]JSONRPCRequest, 0),
		callCount: 0,
	}

	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.mu.Lock()
		defer rs.mu.Unlock()

		rs.callCount++
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			rs.requests = append(rs.requests, req)
		}

		w.WriteHeader(statusCode)
		if rs.response != "" {
			w.Write([]byte(rs.response)) // nolint
		}
	}))

	return rs
}

func (rs *recordingServer) getRequests() []JSONRPCRequest {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	result := make([]JSONRPCRequest, len(rs.requests))
	copy(result, rs.requests)
	return result
}

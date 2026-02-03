package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tender-barbarian/gniot/repository/models"
)

type mockDeviceRepo struct {
	device *models.Device
	err    error
}

func (m *mockDeviceRepo) Create(ctx context.Context, model *models.Device) (int, error) {
	return 0, nil
}
func (m *mockDeviceRepo) Get(ctx context.Context, id int) (*models.Device, error) {
	return m.device, m.err
}
func (m *mockDeviceRepo) GetAll(ctx context.Context) ([]*models.Device, error) {
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

type mockActionRepo struct {
	action *models.Action
	err    error
}

func (m *mockActionRepo) Create(ctx context.Context, model *models.Action) (int, error) {
	return 0, nil
}
func (m *mockActionRepo) Get(ctx context.Context, id int) (*models.Action, error) {
	return m.action, m.err
}
func (m *mockActionRepo) GetAll(ctx context.Context) ([]*models.Action, error) {
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

func TestExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("validation errors", func(t *testing.T) {
		tests := []struct {
			name       string
			deviceRepo *mockDeviceRepo
			actionRepo *mockActionRepo
			wantErr    string
		}{
			{
				name:       "device not found",
				deviceRepo: &mockDeviceRepo{err: errors.New("not found")},
				actionRepo: &mockActionRepo{},
				wantErr:    "getting device: not found",
			},
			{
				name:       "action not found",
				deviceRepo: &mockDeviceRepo{device: &models.Device{ID: 1, Actions: "[1]"}},
				actionRepo: &mockActionRepo{err: errors.New("not found")},
				wantErr:    "getting action: not found",
			},
			{
				name:       "action does not belong to device",
				deviceRepo: &mockDeviceRepo{device: &models.Device{ID: 1, Actions: "[2,3]"}},
				actionRepo: &mockActionRepo{action: &models.Action{ID: 1}},
				wantErr:    "action 1 does not belong to device 1",
			},
			{
				name:       "device unreachable",
				deviceRepo: &mockDeviceRepo{device: &models.Device{ID: 1, IP: "127.0.0.1:99999", Actions: "[1]"}},
				actionRepo: &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: `{}`}},
				wantErr:    "calling device: Post \"http://127.0.0.1:99999/rpc\": dial tcp: address 99999: invalid port",
			},
			{
				name:       "public IP rejected",
				deviceRepo: &mockDeviceRepo{device: &models.Device{ID: 1, IP: "8.8.8.8:80", Actions: "[1]"}},
				actionRepo: &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: `{}`}},
				wantErr:    "device IP must be in private range",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc := NewService(tt.deviceRepo, tt.actionRepo, &mockJobRepo{}, &slog.Logger{})
				err := svc.Execute(ctx, 1, 1)
				assert.EqualError(t, err, tt.wantErr)
			})
		}
	})

	t.Run("mock device execution", func(t *testing.T) {
		tests := []struct {
			name         string
			serverStatus int
			actionPath   string
			actionParams string
			wantErr      string
			validateReq  func(t *testing.T, req JSONRPCRequest)
		}{
			{
				name:         "successful execution",
				serverStatus: http.StatusOK,
				actionPath:   "toggle",
				actionParams: `{"pin":5}`,
				wantErr:      "",
				validateReq: func(t *testing.T, req JSONRPCRequest) {
					assert.Equal(t, "2.0", req.JSONRPC)
					assert.Equal(t, "toggle", req.Method)
					assert.Equal(t, map[string]any{"pin": float64(5)}, req.Params)
				},
			},
			{
				name:         "successful execution with empty params",
				serverStatus: http.StatusOK,
				actionPath:   "status",
				actionParams: "",
				wantErr:      "",
				validateReq: func(t *testing.T, req JSONRPCRequest) {
					assert.Nil(t, req.Params)
				},
			},
			{
				name:         "device returns non-200 status",
				serverStatus: http.StatusInternalServerError,
				actionPath:   "toggle",
				actionParams: `{}`,
				wantErr:      "device returned status 500",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.validateReq != nil {
						var req JSONRPCRequest
						if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
							t.Fatal(err)
						}
						tt.validateReq(t, req)
					}
					w.WriteHeader(tt.serverStatus)
				}))
				defer server.Close()

				deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: server.Listener.Addr().String(), Actions: "[1,2]"}}
				actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: tt.actionPath, Params: tt.actionParams}}
				svc := NewService(deviceRepo, actionRepo, nil, &slog.Logger{})

				err := svc.Execute(ctx, 1, 1)

				if tt.wantErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, tt.wantErr)
				}
			})
		}
	})
}

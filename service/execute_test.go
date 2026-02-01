package service

import (
	"context"
	"encoding/json"
	"errors"
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
	t.Run("device not found", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{err: errors.New("not found")}
		actionRepo := &mockActionRepo{}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "getting device: not found")
	})

	t.Run("action not found", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, Actions: "[1]"}}
		actionRepo := &mockActionRepo{err: errors.New("not found")}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "getting action: not found")
	})

	t.Run("invalid device actions JSON", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, Actions: "invalid"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "unmarshalling device actions: invalid character 'i' looking for beginning of value")
	})

	t.Run("action does not belong to device", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, Actions: "[2,3]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "action 1 does not belong to device 1")
	})

	t.Run("successful execution", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rpc", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req JSONRPCRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "2.0", req.JSONRPC)
			assert.Equal(t, "toggle", req.Method)
			assert.Equal(t, map[string]interface{}{"pin": float64(5)}, req.Params)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ip := server.Listener.Addr().String()
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: ip, Actions: "[1,2]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: `{"pin":5}`}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("successful execution with empty params", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req JSONRPCRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, req.Params)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ip := server.Listener.Addr().String()
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: ip, Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "status", Params: ""}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("device returns non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		ip := server.Listener.Addr().String()
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: ip, Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: `{}`}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "device returned status 500")
	})

	t.Run("device unreachable", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: "127.0.0.1:99999", Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: `{}`}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.ErrorContains(t, err, "calling device")
	})

	t.Run("invalid action params JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ip := server.Listener.Addr().String()
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: ip, Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: "invalid json"}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "parsing action params: invalid character 'i' looking for beginning of value")
	})

	t.Run("public IP rejected", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: "8.8.8.8:80", Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: `{}`}}
		service := NewService(deviceRepo, actionRepo)

		err := service.Execute(ctx, 1, 1)

		assert.EqualError(t, err, "device IP must be in private range")
	})
}

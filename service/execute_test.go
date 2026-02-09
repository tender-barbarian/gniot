package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniotek/cache"
	"github.com/tender-barbarian/gniotek/repository/models"
)

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
				svc := NewService(ServiceConfig{
					DevicesRepo:  tt.deviceRepo,
					ActionsRepo:  tt.actionRepo,
					QueryRepo:    &mockQuerier{},
					DevicesCache: cache.NewCache[*models.Device](),
					ActionsCache: cache.NewCache[*models.Action](),
				})
				_, err := svc.Execute(ctx, 1, 1)
				assert.EqualError(t, err, tt.wantErr)
			})
		}
	})

	t.Run("mock device execution", func(t *testing.T) {
		tests := []struct {
			name           string
			serverStatus   int
			deviceResponse string
			actionPath     string
			actionParams   string
			wantErr        string
			validateReq    func(t *testing.T, req JSONRPCRequest)
		}{
			{
				name:           "successful execution",
				serverStatus:   http.StatusOK,
				deviceResponse: `{"jsonrpc":"2.0","result":{"ok":true},"id":1}`,
				actionPath:     "toggle",
				actionParams:   `{"pin":5}`,
				wantErr:        "",
				validateReq: func(t *testing.T, req JSONRPCRequest) {
					assert.Equal(t, "2.0", req.JSONRPC)
					assert.Equal(t, "toggle", req.Method)
					assert.Equal(t, map[string]any{"pin": float64(5)}, req.Params)
				},
			},
			{
				name:           "successful execution with empty params",
				serverStatus:   http.StatusOK,
				deviceResponse: `{"jsonrpc":"2.0","result":null,"id":1}`,
				actionPath:     "status",
				actionParams:   "",
				wantErr:        "",
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
			{
				name:           "device returns JSON-RPC error",
				serverStatus:   http.StatusOK,
				deviceResponse: `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":1}`,
				actionPath:     "toggle",
				actionParams:   `{}`,
				wantErr:        "JSON-RPC error -32600: Invalid Request",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := createRecordingServer(tt.deviceResponse, tt.serverStatus)
				defer server.Close()

				deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: server.Listener.Addr().String(), Actions: "[1,2]"}}
				actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: tt.actionPath, Params: tt.actionParams}}
				svc := NewService(ServiceConfig{
					DevicesRepo:  deviceRepo,
					ActionsRepo:  actionRepo,
					QueryRepo:    &mockQuerier{},
					DevicesCache: cache.NewCache[*models.Device](),
					ActionsCache: cache.NewCache[*models.Action](),
				})

				_, err := svc.Execute(ctx, 1, 1)

				if tt.wantErr == "" {
					require.NoError(t, err)
				} else {
					assert.EqualError(t, err, tt.wantErr)
				}

				if tt.validateReq != nil {
					reqs := server.getRequests()
					assert.Len(t, reqs, 1)
					tt.validateReq(t, reqs[0])
				}
			})
		}
	})
}

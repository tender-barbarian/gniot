package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tender-barbarian/gniot/repository/models"
)

type mockJobRepo struct {
	jobs    []*models.Job
	err     error
	updated *models.Job
}

func (m *mockJobRepo) Create(ctx context.Context, model *models.Job) (int, error) {
	return 0, m.err
}
func (m *mockJobRepo) Get(ctx context.Context, id int) (*models.Job, error) {
	return nil, m.err
}
func (m *mockJobRepo) GetAll(ctx context.Context) ([]*models.Job, error) {
	return m.jobs, m.err
}
func (m *mockJobRepo) Delete(ctx context.Context, id int) error {
	return m.err
}
func (m *mockJobRepo) Update(ctx context.Context, model *models.Job, id int) error {
	m.updated = model
	return m.err
}
func (m *mockJobRepo) GetTable() string {
	return "jobs"
}

func TestProcessJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	t.Run("test job processing", func(t *testing.T) {
		tests := []struct {
			name         string
			runAt        string
			validateReq  func(t *testing.T, req JSONRPCRequest)
			deviceErr    error
			jobErr       error
			wantErr      string
			expectUpdate bool
		}{
			{
				name:  "job executed, runAt rescheduled",
				runAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				validateReq: func(t *testing.T, req JSONRPCRequest) {
					assert.Equal(t, "2.0", req.JSONRPC)
					assert.Equal(t, "toggle", req.Method)
					assert.Equal(t, map[string]any{"pin": float64(5)}, req.Params)
				},
				expectUpdate: true,
			},
			{
				name:         "device not found, runAt rescheduled",
				runAt:        time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				deviceErr:    errors.New("device not found"),
				expectUpdate: true,
			},
			{
				name:         "job skipped",
				runAt:        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				expectUpdate: false,
			},
			{
				name:         "error getting jobs",
				jobErr:       errors.New("db error"),
				wantErr:      "getting jobs: db error",
				expectUpdate: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockDevice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.validateReq != nil {
						var req JSONRPCRequest
						if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
							t.Fatal(err)
						}
						tt.validateReq(t, req)
					}
					w.WriteHeader(http.StatusOK)
				}))
				defer mockDevice.Close()

				jobRepo := &mockJobRepo{
					jobs: []*models.Job{
						{ID: 1, Devices: "[1]", Action: "1", RunAt: tt.runAt, Interval: "1h"},
					},
					err: tt.jobErr,
				}
				deviceRepo := &mockDeviceRepo{
					device: &models.Device{
						ID: 1, IP: mockDevice.Listener.Addr().String(), Actions: "[1]",
					},
					err: tt.deviceErr,
				}
				actionRepo := &mockActionRepo{
					action: &models.Action{
						ID: 1, Path: "toggle", Params: `{"pin":5}`,
					},
				}

				svc := NewService(deviceRepo, actionRepo, jobRepo)

				err := svc.processJobs(ctx, logger)
				if tt.wantErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, tt.wantErr)
				}

				if tt.expectUpdate {
					assert.NotNil(t, jobRepo.updated)
					updatedTime, _ := time.Parse(time.RFC3339, jobRepo.updated.RunAt)
					assert.True(t, updatedTime.After(time.Now().Add(59*time.Minute)))
				} else {
					assert.Nil(t, jobRepo.updated)
				}
			})
		}
	})
}

func TestRunJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		jobRepo := &mockJobRepo{jobs: []*models.Job{}}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)
		errCh := make(chan error, 10)

		done := make(chan struct{})
		go func() {
			svc.RunJobs(ctx, logger, 10*time.Millisecond, errCh)
			close(done)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("RunJobs did not stop after context cancellation")
		}
	})

	t.Run("sends errors to channel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		jobRepo := &mockJobRepo{err: errors.New("db error")}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)
		errCh := make(chan error, 10)

		go svc.RunJobs(ctx, logger, 10*time.Millisecond, errCh)

		select {
		case err := <-errCh:
			assert.Contains(t, err.Error(), "db error")
		case <-time.After(1 * time.Second):
			t.Fatal("expected error on channel")
		}
	})
}

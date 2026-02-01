package service

import (
	"context"
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

type mockJobRepoWithJobs struct {
	jobs      []*models.Job
	getAllErr error
	updateErr error
	updated   *models.Job
}

func (m *mockJobRepoWithJobs) Create(ctx context.Context, model *models.Job) (int, error) {
	return 0, nil
}
func (m *mockJobRepoWithJobs) Get(ctx context.Context, id int) (*models.Job, error) {
	return nil, nil
}
func (m *mockJobRepoWithJobs) GetAll(ctx context.Context) ([]*models.Job, error) {
	return m.jobs, m.getAllErr
}
func (m *mockJobRepoWithJobs) Delete(ctx context.Context, id int) error {
	return nil
}
func (m *mockJobRepoWithJobs) Update(ctx context.Context, model *models.Job, id int) error {
	m.updated = model
	return m.updateErr
}
func (m *mockJobRepoWithJobs) GetTable() string {
	return "jobs"
}

func TestProcessJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	t.Run("job in past gets executed and rescheduled", func(t *testing.T) {
		mockDevice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer mockDevice.Close()

		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "[1]", Action: "1", RunAt: pastTime, Interval: "1h"},
			},
		}
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: mockDevice.Listener.Addr().String(), Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: "{}"}}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, jobRepo.updated)
		updatedTime, _ := time.Parse(time.RFC3339, jobRepo.updated.RunAt)
		assert.True(t, updatedTime.After(time.Now().Add(59*time.Minute)))
	})

	t.Run("job in future is skipped", func(t *testing.T) {
		futureTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "[1]", Action: "1", RunAt: futureTime, Interval: "1h"},
			},
		}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, jobRepo.updated)
	})

	t.Run("error getting jobs", func(t *testing.T) {
		jobRepo := &mockJobRepoWithJobs{getAllErr: errors.New("db error")}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		assert.EqualError(t, err, "getting jobs: db error")
	})

	t.Run("invalid time format", func(t *testing.T) {
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "[1]", Action: "1", RunAt: "invalid", Interval: "1h"},
			},
		}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, jobRepo.updated)
	})

	t.Run("invalid devices JSON", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "invalid", Action: "1", RunAt: pastTime, Interval: "1h"},
			},
		}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, jobRepo.updated)
	})

	t.Run("invalid action ID", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "[1]", Action: "abc", RunAt: pastTime, Interval: "1h"},
			},
		}
		deviceRepo := &mockDeviceRepo{}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, jobRepo.updated)
	})

	t.Run("invalid interval", func(t *testing.T) {
		mockDevice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer mockDevice.Close()

		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "[1]", Action: "1", RunAt: pastTime, Interval: "invalid"},
			},
		}
		deviceRepo := &mockDeviceRepo{device: &models.Device{ID: 1, IP: mockDevice.Listener.Addr().String(), Actions: "[1]"}}
		actionRepo := &mockActionRepo{action: &models.Action{ID: 1, Path: "toggle", Params: "{}"}}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, jobRepo.updated)
	})

	t.Run("execute error logs but still reschedules", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		jobRepo := &mockJobRepoWithJobs{
			jobs: []*models.Job{
				{ID: 1, Devices: "[1]", Action: "1", RunAt: pastTime, Interval: "1h"},
			},
		}
		deviceRepo := &mockDeviceRepo{err: errors.New("device not found")}
		actionRepo := &mockActionRepo{}
		svc := NewService(deviceRepo, actionRepo, jobRepo)

		err := svc.processJobs(ctx, logger)

		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, jobRepo.updated)
	})
}

func TestRunJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		jobRepo := &mockJobRepoWithJobs{jobs: []*models.Job{}}
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

		jobRepo := &mockJobRepoWithJobs{getAllErr: errors.New("db error")}
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

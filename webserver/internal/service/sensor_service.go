package service

import (
	"context"
	"database/sql"

	"github.com/tender-barbarian/gniot/webserver/internal/config"
	"github.com/tender-barbarian/gniot/webserver/internal/model"
	"github.com/tender-barbarian/gniot/webserver/internal/repository"
)

// SensorService is the service to manage the Sensors.
type SensorService struct {
	config     *config.Config
	repository *repository.SensorRepository
}

// NewSensorService returns a new NewSensorService.
func NewSensorService(config *config.Config, repository *repository.SensorRepository) *SensorService {
	return &SensorService{
		config:     config,
		repository: repository,
	}
}

// List returns a list of all Sensors, filterable by name and job.
func (s *SensorService) List(ctx context.Context, name string, sensorType string, chip string, board string) ([]model.Sensor, error) {
	return s.repository.FindAll(ctx, repository.SensorRepositoryFindAllParams{
		Name:       sql.NullString{String: name, Valid: name != ""},
		SensorType: sql.NullString{String: sensorType, Valid: sensorType != ""},
		Chip:       sql.NullString{String: chip, Valid: chip != ""},
		Board:      sql.NullString{String: board, Valid: board != ""},
	})
}

// Create creates a new Sensor.
func (s *SensorService) Create(ctx context.Context, name string, sensorType string, chip string, board string) (int, error) {
	return s.repository.Create(ctx, repository.SensorRepositoryCreateParams{
		Name:       name,
		SensorType: sensorType,
		Chip:       chip,
		Board:      board,
	})
}

// Get returns a Sensor by id.
func (s *SensorService) Get(ctx context.Context, id int) (model.Sensor, error) {
	return s.repository.Find(ctx, id)
}

// Delete deletes a Sensor by id.
func (s *SensorService) Delete(ctx context.Context, id int) error {
	return s.repository.Delete(ctx, id)
}

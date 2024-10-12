package service

import (
	"context"

	"github.com/tender-barbarian/gniot/webserver/internal/config"
	"github.com/tender-barbarian/gniot/webserver/internal/model"
	"github.com/tender-barbarian/gniot/webserver/internal/repository"
)

// SensorMethodService is the service to manage the SensorMethods.
type SensorMethodService struct {
	config     *config.Config
	repository *repository.SensorMethodRepository
}

// NewSensorMethodService returns a new NewSensorMethodService.
func NewSensorMethodService(config *config.Config, repository *repository.SensorMethodRepository) *SensorMethodService {
	return &SensorMethodService{
		config:     config,
		repository: repository,
	}
}

// List returns a list of all SensorMethods, filterable by name and job.
func (s *SensorMethodService) List(ctx context.Context, sensorMethodIDs []int32) ([]model.SensorMethod, error) {
	return s.repository.FindAll(ctx, sensorMethodIDs)
}

// Create creates a new SensorMethod.
func (s *SensorMethodService) Create(ctx context.Context, name string, HttpMethod string, RequestBody string, board string) (int, error) {
	return s.repository.Create(ctx, repository.SensorMethodRepositoryCreateParams{
		Name:        name,
		HttpMethod:  HttpMethod,
		RequestBody: RequestBody,
	})
}

// Get returns a SensorMethod by id.
func (s *SensorMethodService) Get(ctx context.Context, id int) (model.SensorMethod, error) {
	return s.repository.Find(ctx, id)
}

// Delete deletes a SensorMethod by id.
func (s *SensorMethodService) Delete(ctx context.Context, id int) error {
	return s.repository.Delete(ctx, id)
}

package service

import (
	"github.com/tender-barbarian/gniot/repository"
	"github.com/tender-barbarian/gniot/repository/models"
)

type Service struct {
	devicesRepo repository.GenericRepo[*models.Device]
	actionsRepo repository.GenericRepo[*models.Action]
	jobsRepo    repository.GenericRepo[*models.Job]
}

func NewService(devicesRepo repository.GenericRepo[*models.Device], actionsRepo repository.GenericRepo[*models.Action], jobsRepo repository.GenericRepo[*models.Job]) *Service {
	return &Service{
		devicesRepo: devicesRepo,
		actionsRepo: actionsRepo,
		jobsRepo:    jobsRepo,
	}
}

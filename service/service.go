package service

import (
	"github.com/tender-barbarian/gniot/repository"
	gocrud "github.com/tender-barbarian/go-crud"
)

type Service[M, N, O gocrud.Model] struct {
	devicesRepo repository.GenericRepo[M]
	actionsRepo repository.GenericRepo[N]
	jobsRepo    repository.GenericRepo[O]
}

func NewService[M, N, O gocrud.Model](devicesRepo repository.GenericRepo[M], actionsRepo repository.GenericRepo[N], jobsRepo repository.GenericRepo[O]) *Service[M, N, O] {
	return &Service[M, N, O]{
		devicesRepo: devicesRepo,
		actionsRepo: actionsRepo,
		jobsRepo:    jobsRepo,
	}
}

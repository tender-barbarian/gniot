package service

import (
	"github.com/tender-barbarian/gniot/repository"
	gocrud "github.com/tender-barbarian/go-crud"
)

type Service[M, N gocrud.Model] struct {
	devicesRepo repository.GenericRepo[M]
	actionsRepo repository.GenericRepo[N]
}

func NewService[M, N gocrud.Model](devicesRepo repository.GenericRepo[M], actionsRepo repository.GenericRepo[N]) *Service[M, N] {
	return &Service[M, N]{
		devicesRepo: devicesRepo,
		actionsRepo: actionsRepo,
	}
}

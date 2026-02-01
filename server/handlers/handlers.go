package handlers

import (
	"github.com/tender-barbarian/gniot/service"
	gocrud "github.com/tender-barbarian/go-crud"
)

type Handlers[M, N, O gocrud.Model] struct {
	errorHandler *ErrorHandler
	service      *service.Service[M, N, O]
}

func NewCustomHandlers[M, N, O gocrud.Model](errorHandler *ErrorHandler, service *service.Service[M, N, O]) *Handlers[M, N, O] {
	return &Handlers[M, N, O]{
		errorHandler: errorHandler,
		service:      service,
	}
}

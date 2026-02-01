package handlers

import (
	"github.com/tender-barbarian/gniot/service"
	gocrud "github.com/tender-barbarian/go-crud"
)

type Handlers[M, N gocrud.Model] struct {
	errorHandler *ErrorHandler
	service *service.Service[M, N]
}

func NewCustomHandlers[M, N gocrud.Model](errorHandler *ErrorHandler, service *service.Service[M, N]) *Handlers[M, N] {
	return &Handlers[M, N]{
		errorHandler: errorHandler,
		service: service,
	}
}

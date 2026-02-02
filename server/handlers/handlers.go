package handlers

import (
	"log/slog"

	"github.com/tender-barbarian/gniot/service"
)

type CustomHandlers struct {
	logger  *slog.Logger
	service *service.Service
}

func NewCustomHandlers(logger *slog.Logger, service *service.Service) *CustomHandlers {
	return &CustomHandlers{
		logger:  logger,
		service: service,
	}
}

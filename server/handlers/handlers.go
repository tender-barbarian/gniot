package handlers

import (
	"log/slog"

	"github.com/tender-barbarian/gniot/service"
)

type CustomHandlers struct {
	logger  *slog.Logger
	service *service.Service
	*ErrorHandler
}

func NewCustomHandlers(logger *slog.Logger, service *service.Service, eh *ErrorHandler) *CustomHandlers {
	return &CustomHandlers{
		logger:       logger,
		service:      service,
		ErrorHandler: eh,
	}
}

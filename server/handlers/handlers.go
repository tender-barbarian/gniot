package handlers

import (
	"context"
	"log/slog"

	"github.com/tender-barbarian/gniotek/service"
)

type Executor interface {
	Execute(ctx context.Context, deviceId, actionId int) (*service.JSONRPCResponse, error)
}

type CustomHandlers struct {
	logger  *slog.Logger
	service Executor
	*ErrorHandler
}

func NewCustomHandlers(logger *slog.Logger, service Executor, eh *ErrorHandler) *CustomHandlers {
	return &CustomHandlers{
		logger:       logger,
		service:      service,
		ErrorHandler: eh,
	}
}

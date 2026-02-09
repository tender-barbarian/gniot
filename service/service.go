package service

import (
	"log/slog"
	"sync"

	"github.com/tender-barbarian/gniotek/cache"
	"github.com/tender-barbarian/gniotek/repository"
	"github.com/tender-barbarian/gniotek/repository/models"
)

type ServiceConfig struct {
	DevicesRepo     repository.GenericRepo[*models.Device]
	ActionsRepo     repository.GenericRepo[*models.Action]
	AutomationsRepo repository.GenericRepo[*models.Automation]
	QueryRepo       repository.Querier
	DevicesCache    *cache.Cache[*models.Device]
	ActionsCache    *cache.Cache[*models.Action]
	Logger          *slog.Logger
}

type Service struct {
	devicesRepo     repository.GenericRepo[*models.Device]
	actionsRepo     repository.GenericRepo[*models.Action]
	automationsRepo repository.GenericRepo[*models.Automation]
	queryRepo       repository.Querier
	devicesCache    *cache.Cache[*models.Device]
	actionsCache    *cache.Cache[*models.Action]
	logger          *slog.Logger
	deviceMu        sync.Map
}

func NewService(cfg ServiceConfig) *Service {
	return &Service{
		devicesRepo:     cfg.DevicesRepo,
		actionsRepo:     cfg.ActionsRepo,
		automationsRepo: cfg.AutomationsRepo,
		queryRepo:       cfg.QueryRepo,
		devicesCache:    cfg.DevicesCache,
		actionsCache:    cfg.ActionsCache,
		logger:          cfg.Logger,
	}
}

package server

import (
	"context"
	"net/http"

	"github.com/tender-barbarian/gniot/webserver/internal/handler/sensor"
	"github.com/tender-barbarian/gniot/webserver/internal/logging"
	"github.com/tender-barbarian/gniot/webserver/internal/repository"
	"github.com/tender-barbarian/gniot/webserver/internal/router"
	"github.com/tender-barbarian/gniot/webserver/internal/service"
)

func NewServer(ctx context.Context) http.Handler {
	mux := http.NewServeMux()

	sensorRepository := repository.NewSensorRepository(nil)
	sensorService := service.NewSensorService(nil, sensorRepository)
	getSensor := sensor.NewGetSensorHandler(sensorService)

	sensorMethodRepository := repository.NewSensorMethodRepository(nil)
	sensorMethodService := service.NewSensorMethodService(nil, sensorMethodRepository)
	execSensorMethod := sensor.NewExecSensorMethodHandler(sensorService, sensorMethodService)

	router.AddRoutes(ctx, mux, getSensor, execSensorMethod)

	var handler http.Handler = mux
	handler = logging.NewLoggingMiddleware(handler)

	return handler
}

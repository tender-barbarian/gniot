package router

import (
	"context"
	"net/http"

	"github.com/tender-barbarian/gniot/webserver/internal/handler/sensor"
)

func AddRoutes(ctx context.Context, mux *http.ServeMux, getSensor *sensor.GetSensorHandler, execSensorMethod *sensor.ExecSensorMethodHandler) {
	mux.HandleFunc("/sensor/{id}", getSensor.Handle(ctx))
	mux.HandleFunc("/sensor/{id}/{sensor_method}", execSensorMethod.Handle(ctx))
	mux.Handle("/", http.NotFoundHandler())
}

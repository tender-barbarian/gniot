package sensor

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tender-barbarian/gniot/webserver/internal/service"
)

// ExecSensorMethodHandler is the http handler to get a Sensor.
type ExecSensorMethodHandler struct {
	sensorService       *service.SensorService
	sensorMethodService *service.SensorMethodService
}

// NewExecSensorMethodHandler returns a new ExecSensorMethodHandler.
func NewExecSensorMethodHandler(sensorService *service.SensorService, sensorMethodService *service.SensorMethodService) *ExecSensorMethodHandler {
	return &ExecSensorMethodHandler{
		sensorService:       sensorService,
		sensorMethodService: sensorMethodService,
	}
}

// Handle handles the http request.
func (h *ExecSensorMethodHandler) Handle(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sensorId, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid Sensor id: %v", err), http.StatusBadRequest)
			return
		}

		sensor, err := h.sensorService.Get(ctx, sensorId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, fmt.Sprintf("cannot find Sensor with id %d: %v", sensorId, err), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sensorMethods, err := h.sensorMethodService.List(ctx, sensor.SensorMethodIDs)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, fmt.Sprintf("Sensor with id %d has no methods: %v", sensorId, err), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, sensorMethod := range sensorMethods {
			if r.PathValue("sensor_method") == sensorMethod.Name {
				r, err = http.NewRequestWithContext(ctx, sensorMethod.HttpMethod, sensor.IP, bytes.NewBuffer([]byte(sensorMethod.RequestBody)))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				r.Header.Add("Content-Type", "application/json")

				client := &http.Client{}
				res, err := client.Do(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				defer res.Body.Close()

				w.WriteHeader(http.StatusOK)

				// TODO: return whatever sensor sent back
			}
		}
	}
}

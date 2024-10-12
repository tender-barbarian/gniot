package sensor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tender-barbarian/gniot/webserver/internal/service"
)

// GetSensorHandler is the http handler to get a Sensor.
type GetSensorHandler struct {
	service *service.SensorService
}

// NewGetSensorHandler returns a new GetSensorHandler.
func NewGetSensorHandler(service *service.SensorService) *GetSensorHandler {
	return &GetSensorHandler{
		service: service,
	}
}

// Handle handles the http request.
func (h *GetSensorHandler) Handle(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sensorId, err := strconv.Atoi(r.URL.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid Sensor id: %v", err), http.StatusBadRequest)
			return
		}

		sensor, err := h.service.Get(ctx, sensorId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, fmt.Sprintf("cannot find Sensor with id %d: %v", sensorId, err), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response, err := json.Marshal(sensor)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		_, err = w.Write(response)
		if err != nil {
			fmt.Print(err)
		}
	}
}

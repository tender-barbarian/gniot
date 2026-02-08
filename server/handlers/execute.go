package handlers

import (
	"encoding/json"
	"net/http"
)

type ExecuteReqBody struct {
	DeviceId *int `json:"deviceId"`
	ActionId *int `json:"actionId"`
}

func (h *CustomHandlers) Execute(w http.ResponseWriter, r *http.Request) {
	var e ExecuteReqBody
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		h.WriteError(w, r, err, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if e.DeviceId == nil || e.ActionId == nil {
		h.WriteError(w, r, nil, "invalid params", http.StatusBadRequest)
		return
	}

	deviceResponse, err := h.service.Execute(r.Context(), *e.DeviceId, *e.ActionId)
	if err != nil {
		h.WriteError(w, r, err, "job failed to execute", http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(deviceResponse)
	if err != nil {
		h.WriteError(w, r, err, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buf)
	if err != nil {
		h.logger.Error("failed to write output", "error", err)
		return
	}
}

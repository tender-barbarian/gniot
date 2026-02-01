package handlers

import (
	"encoding/json"
	"net/http"
)

type ExecuteReqBody struct {
	DeviceId *int `json:"deviceId"`
	ActionId *int `json:"actionId"`
}

func (h *Handlers[M, N]) Execute(w http.ResponseWriter, r *http.Request) {
    var e ExecuteReqBody
    err := json.NewDecoder(r.Body).Decode(&e)
    if err != nil {
        h.errorHandler.WriteError(w, r, err, "invalid JSON body", http.StatusBadRequest)
		return
    }

	if e.DeviceId == nil || e.ActionId == nil {
		h.errorHandler.WriteError(w, r, nil, "invalid params", http.StatusBadRequest)
		return
	}

    err = h.service.Execute(r.Context(), *e.DeviceId, *e.ActionId)
	if err != nil {
        h.errorHandler.WriteError(w, r, err, "job failed to execute", http.StatusInternalServerError)
		return
    }
}

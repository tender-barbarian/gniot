package handlers

import "net/http"

func (h *CustomHandlers) WriteError(w http.ResponseWriter, r *http.Request, err error, msg string, statusCode int) {
	if err == nil {
		h.logger.Error(msg, "method", r.Method, "uri", r.URL.RequestURI())
	} else {
		h.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
	}

	http.Error(w, msg, statusCode)
}

package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func respondFail(r *http.Request, w http.ResponseWriter, code int, msg string, err error) {
	slog.ErrorContext(r.Context(), "http request execution failed",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Int("http_status", code),
		slog.Any("error", err),
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal json in response", "error", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

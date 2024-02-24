package httpserver

import (
	"context"
	"net/http"
)

type Handlers struct {
	service ConfigurationService
}

func newHandlers(s ConfigurationService) *Handlers {
	return &Handlers{s}
}

func (e Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	err := e.service.HealthCheck(context.Background())
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (e Handlers) ReadinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	err := e.service.HealthCheck(context.Background())
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

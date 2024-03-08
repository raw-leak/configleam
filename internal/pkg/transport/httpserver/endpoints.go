package httpserver

import (
	"net/http"
)

type Handlers struct {
	service ConfigurationService
}

func newHandlers(s ConfigurationService) *Handlers {
	return &Handlers{s}
}

func (e Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// TODO

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (e Handlers) ReadinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	// TODO

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

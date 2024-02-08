package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/raw-leak/configleam/internal/app/configleam"
)

type ConfigleamService interface {
	HealthCheck(ctx context.Context) error
}

type ConfigleamEndpoints interface {
	CloneConfigHandler(w http.ResponseWriter, r *http.Request)
	ReadConfigurationHandler(w http.ResponseWriter, r *http.Request)
}

type ConfigleamSet interface {
	ConfigleamService
	ConfigleamEndpoints
}

type httpServer struct {
	server     *http.Server
	configleam *configleam.ConfigleamSet
}

func NewHttpTransport(configleam *configleam.ConfigleamSet) *httpServer {
	return &httpServer{configleam: configleam}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	endpoints := newHandlers(s.configleam.Service)

	// register health and readiness handlers
	mux.HandleFunc("/health", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// business handlers
	mux.HandleFunc("/v1/cfg", s.configleam.Endpoints.ReadConfigurationHandler)
	mux.HandleFunc("/v1/cfg/clone", s.configleam.Endpoints.CloneConfigHandler)

	s.server = &http.Server{Addr: fmt.Sprintf(":%s", httpAddr), Handler: mux}

	log.Printf("Starting HTTP server on port %s\n", httpAddr)
	return s.server.ListenAndServe()
}

func (t *httpServer) Shutdown(ctx context.Context) error {
	if t.server != nil {
		return t.server.Shutdown(ctx)
	}
	return nil
}

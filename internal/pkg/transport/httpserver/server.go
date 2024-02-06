package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type Service interface {
	ReadConfig(ctx context.Context, env string, groups, globals []string) (map[string]interface{}, error)
	HealthCheck(ctx context.Context) error
}

type httpServer struct {
	server  *http.Server
	service Service
}

func NewHttpTransport(service Service) *httpServer {
	return &httpServer{service: service}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	endpoints := NewEndpoints(s.service)

	// register health and readiness handlers
	mux.HandleFunc("/healthz", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// business handlers
	mux.HandleFunc("/v1/cfg", endpoints.ReadConfigurationHandler)

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

package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type httpServer struct {
	server *http.Server
}

func NewHttpTransport() *httpServer {
	return &httpServer{}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	// register health and readiness handlers
	mux.HandleFunc("/healthz", healthCheckHandler)
	mux.HandleFunc("/ready", readinessCheckHandler)

	mux.HandleFunc("/cfg", readConfigurationHandler)

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

package httpserver

import (
	"context"
	"fmt"
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

	s.server = &http.Server{Addr: fmt.Sprintf(":%s", httpAddr), Handler: mux}

	return s.server.ListenAndServe()
}

func (t *httpServer) Shutdown(ctx context.Context) error {
	if t.server != nil {
		return t.server.Shutdown(ctx)
	}
	return nil
}

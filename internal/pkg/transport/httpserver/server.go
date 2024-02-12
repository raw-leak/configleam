package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type ConfigleamSet interface {
	ConfigleamService
	ConfigleamEndpoints
}

type ConfigleamService interface {
	HealthCheck(ctx context.Context) error
}

type ConfigleamEndpoints interface {
	CloneConfigHandler(w http.ResponseWriter, r *http.Request)
	ReadConfigHandler(w http.ResponseWriter, r *http.Request)
	DeleteConfigHandler(w http.ResponseWriter, r *http.Request)
}

type ConfigleamSecretsSet interface {
	ConfigleamSecretsService
	ConfigleamSecretsEndpoints
}

type ConfigleamSecretsEndpoints interface {
	UpsertSecretsHandler(w http.ResponseWriter, r *http.Request)
}

type ConfigleamSecretsService interface {
	UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error
}

type httpServer struct {
	server            *http.Server
	configleam        ConfigleamSet
	configleamSecrets ConfigleamSecretsSet
}

func NewHttpTransport(configleam ConfigleamSet, configleamSecrets ConfigleamSecretsSet) *httpServer {
	return &httpServer{configleam: configleam, configleamSecrets: configleamSecrets}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	endpoints := newHandlers(s.configleam)

	// register health and readiness handlers
	mux.HandleFunc("/health", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// configleam repo business handlers
	mux.HandleFunc("/v1/cfg", s.configleam.ReadConfigHandler)
	mux.HandleFunc("/v1/cfg/clone", s.configleam.CloneConfigHandler)
	mux.HandleFunc("/v1/cfg/delete", s.configleam.DeleteConfigHandler)

	// configleam secrets business handlers
	mux.HandleFunc("/v1/secrets", s.configleamSecrets.UpsertSecretsHandler)

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

package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"

	p "github.com/raw-leak/configleam/internal/pkg/permissions"
)

type httpServer struct {
	server *http.Server

	configleam        ConfigleamSet
	configleamSecrets ConfigleamSecretsSet
	configleamAccess  ConfigleamAccessSet

	permissions PermissionsBuilder
}

func NewHttpServer(configleam ConfigleamSet, configleamSecrets ConfigleamSecretsSet, configleamAccess ConfigleamAccessSet, permissions PermissionsBuilder) *httpServer {
	return &httpServer{configleam: configleam, configleamSecrets: configleamSecrets, configleamAccess: configleamAccess, permissions: permissions}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	endpoints := newHandlers(s.configleam)

	// register health and readiness handlers
	mux.HandleFunc("/health", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// middlewares
	auth := NewAuthMiddleware(s.configleamAccess, s.permissions)

	// TODO migrate to go 1.22

	// rad configuration business handlers
	mux.HandleFunc("/v1/config", auth.Guard(p.ReadConfig)(s.configleam.ReadConfigHandler))

	// clone environment business handlers
	mux.HandleFunc("/v1/config/clone", auth.Guard(p.CloneEnvironment)(s.configleam.CloneConfigHandler))
	mux.HandleFunc("/v1/config/clone/delete", auth.Guard(p.CloneEnvironment)(s.configleam.DeleteConfigHandler))

	// secrets business handlers
	mux.HandleFunc("/v1/secrets", auth.Guard(p.CreateSecrets)(s.configleamSecrets.UpsertSecretsHandler))

	// configleam access business handlers
	mux.HandleFunc("/v1/access", auth.Guard(p.Admin)(s.configleamAccess.GenerateAccessKeyHandler))
	mux.HandleFunc("/v1/access/delete", auth.Guard(p.Admin)(s.configleamAccess.DeleteAccessKeysHandler))

	// dashboard business handlers
	// TODO

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

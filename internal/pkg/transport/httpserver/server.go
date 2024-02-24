package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/raw-leak/configleam/internal/pkg/auth"
	p "github.com/raw-leak/configleam/internal/pkg/permissions"
)

type httpServer struct {
	server *http.Server

	configuration ConfigurationSet
	secrets       SecretsSet
	access        AccessSet
	dashboard     DashboardSet

	permissions PermissionsBuilder
}

func NewHttpServer(configuration ConfigurationSet, secrets SecretsSet, access AccessSet, dashboard DashboardSet, permissions PermissionsBuilder) *httpServer {
	return &httpServer{
		configuration: configuration,
		secrets:       secrets,
		access:        access,
		dashboard:     dashboard,
		permissions:   permissions,
	}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	endpoints := newHandlers(s.configuration)

	// register health and readiness handlers
	mux.HandleFunc("/health", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// middlewares
	auth := auth.NewAuthMiddleware(s.access, s.permissions)

	// TODO migrate handlers pattern to go 1.22, by including the HTTP method

	// rad configuration business handlers
	mux.HandleFunc("/v1/config", auth.Guard(p.ReadConfig)(s.configuration.ReadConfigHandler))

	// clone environment business handlers
	mux.HandleFunc("/v1/config/clone", auth.Guard(p.CloneEnvironment)(s.configuration.CloneConfigHandler))
	mux.HandleFunc("/v1/config/clone/delete", auth.Guard(p.CloneEnvironment)(s.configuration.DeleteConfigHandler))

	// secrets business handlers
	mux.HandleFunc("/v1/secrets", auth.Guard(p.CreateSecrets)(s.secrets.UpsertSecretsHandler))

	// configleam access business handlers
	// mux.HandleFunc("/v1/access", auth.Guard(p.Admin)(s.configleamAccess.GenerateAccessKeyHandler))
	mux.HandleFunc("/v1/access", s.access.GenerateAccessKeyHandler)
	mux.HandleFunc("/v1/access/delete", auth.Guard(p.Admin)(s.access.DeleteAccessKeysHandler))

	// dashboard security handlers
	mux.HandleFunc("/v1/dashboard/login", auth.LoginHandler)
	mux.HandleFunc("/v1/dashboard/logout", auth.GuardDashboard()(auth.LogoutHandler))

	// dashboard business handlers
	mux.HandleFunc("/v1/dashboard", auth.GuardDashboard()(s.dashboard.HomeHandler))

	mux.HandleFunc("/v1/dashboard/config", auth.GuardDashboard()(s.dashboard.ConfigHandler))

	mux.HandleFunc("/v1/dashboard/access", auth.GuardDashboard()(s.dashboard.AccessHandler))
	mux.HandleFunc("/v1/dashboard/access/create/params", auth.GuardDashboard()(s.dashboard.CreateAccessKeyParamsHandler))
	mux.HandleFunc("/v1/dashboard/access/create", auth.GuardDashboard()(s.dashboard.CreateAccessKeyHandler))
	mux.HandleFunc("/v1/dashboard/access/delete", auth.GuardDashboard()(s.dashboard.DeleteAccessKeyHandler))

	// serve static
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(path.Join(dir, "static")))))

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

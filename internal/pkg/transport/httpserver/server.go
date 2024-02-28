package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/raw-leak/configleam/internal/pkg/auth"
	"github.com/raw-leak/configleam/internal/pkg/auth/templates"
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
	auth := auth.NewAuthMiddleware(s.access, s.permissions, templates.New())

	// TODO migrate handlers pattern to go 1.22, by including the HTTP method

	// configuration business handlers
	mux.HandleFunc("GET /config", auth.Guard(p.ReadConfig)(s.configuration.ReadConfigHandler))

	// configuration clone environment business handlers
	mux.HandleFunc("POST /config/clone", auth.Guard(p.CloneEnvironment)(s.configuration.CloneConfigHandler))
	mux.HandleFunc("DELETE /config/clone", auth.Guard(p.CloneEnvironment)(s.configuration.DeleteConfigHandler))

	// secrets business handlers
	mux.HandleFunc("PUT /secrets", auth.Guard(p.CreateSecrets)(s.secrets.UpsertSecretsHandler))

	// access business handlers
	mux.HandleFunc("POST /access", s.access.GenerateAccessKeyHandler)
	mux.HandleFunc("DELETE /access", auth.Guard(p.Admin)(s.access.DeleteAccessKeysHandler))

	// dashboard security business handlers
	mux.HandleFunc("/dashboard/login", auth.LoginHandler)
	mux.HandleFunc("/dashboard/logout", auth.GuardDashboard()(auth.LogoutHandler))

	// dashboard business handlers
	mux.HandleFunc("GET /dashboard", auth.GuardDashboard()(s.dashboard.HomeHandler))

	mux.HandleFunc("GET /dashboard/config", auth.GuardDashboard()(s.dashboard.ConfigHandler))

	mux.HandleFunc("GET /dashboard/access", auth.GuardDashboard()(s.dashboard.AccessHandler))
	mux.HandleFunc("GET /dashboard/access/create", auth.GuardDashboard()(s.dashboard.CreateAccessKeyParamsHandler))
	mux.HandleFunc("POST /dashboard/access/create", auth.GuardDashboard()(s.dashboard.CreateAccessKeyHandler))
	mux.HandleFunc("POST /dashboard/access/delete", auth.GuardDashboard()(s.dashboard.DeleteAccessKeyHandler))

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

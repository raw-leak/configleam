package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	configleamdashboard "github.com/raw-leak/configleam/internal/app/configleam-dashboard"

	"github.com/raw-leak/configleam/internal/pkg/auth"
	p "github.com/raw-leak/configleam/internal/pkg/permissions"
)

type httpServer struct {
	server *http.Server

	configleam          ConfigleamSet
	configleamSecrets   ConfigleamSecretsSet
	configleamAccess    ConfigleamAccessSet
	configleamDashboard *configleamdashboard.ConfigleamDashboardSet

	permissions PermissionsBuilder
}

func NewHttpServer(configleam ConfigleamSet, configleamSecrets ConfigleamSecretsSet, configleamAccess ConfigleamAccessSet, configleamDashboard *configleamdashboard.ConfigleamDashboardSet, permissions PermissionsBuilder) *httpServer {
	return &httpServer{
		configleam:          configleam,
		configleamSecrets:   configleamSecrets,
		configleamAccess:    configleamAccess,
		configleamDashboard: configleamDashboard,
		permissions:         permissions,
	}
}

func (s *httpServer) ListenAndServe(httpAddr string) error {
	mux := http.NewServeMux()

	endpoints := newHandlers(s.configleam)

	// register health and readiness handlers
	mux.HandleFunc("/health", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// middlewares
	auth := auth.NewAuthMiddleware(s.configleamAccess, s.permissions)

	// TODO migrate handlers pattern to go 1.22, by including the HTTP method

	// rad configuration business handlers
	mux.HandleFunc("/v1/config", auth.Guard(p.ReadConfig)(s.configleam.ReadConfigHandler))

	// clone environment business handlers
	mux.HandleFunc("/v1/config/clone", auth.Guard(p.CloneEnvironment)(s.configleam.CloneConfigHandler))
	mux.HandleFunc("/v1/config/clone/delete", auth.Guard(p.CloneEnvironment)(s.configleam.DeleteConfigHandler))

	// secrets business handlers
	mux.HandleFunc("/v1/secrets", auth.Guard(p.CreateSecrets)(s.configleamSecrets.UpsertSecretsHandler))

	// configleam access business handlers
	// mux.HandleFunc("/v1/access", auth.Guard(p.Admin)(s.configleamAccess.GenerateAccessKeyHandler))
	mux.HandleFunc("/v1/access", s.configleamAccess.GenerateAccessKeyHandler)
	mux.HandleFunc("/v1/access/delete", auth.Guard(p.Admin)(s.configleamAccess.DeleteAccessKeysHandler))

	// dashboard security handlers
	mux.HandleFunc("/v1/dashboard/login", auth.LoginHandler)
	mux.HandleFunc("/v1/dashboard/logout", auth.GuardDashboard()(auth.LogoutHandler))

	// dashboard business handlers
	mux.HandleFunc("/v1/dashboard", auth.GuardDashboard()(s.configleamDashboard.ConfigleamDashboardEndpoints.HomeHandler))

	mux.HandleFunc("/v1/dashboard/config", auth.GuardDashboard()(s.configleamDashboard.ConfigleamDashboardEndpoints.ConfigHandler))

	mux.HandleFunc("/v1/dashboard/access", auth.GuardDashboard()(s.configleamDashboard.ConfigleamDashboardEndpoints.AccessHandler))
	mux.HandleFunc("/v1/dashboard/access/create/params", auth.GuardDashboard()(s.configleamDashboard.ConfigleamDashboardEndpoints.CreateAccessKeyParamsHandler))
	mux.HandleFunc("/v1/dashboard/access/create", auth.GuardDashboard()(s.configleamDashboard.ConfigleamDashboardEndpoints.CreateAccessKeyHandler))
	mux.HandleFunc("/v1/dashboard/access/delete", auth.GuardDashboard()(s.configleamDashboard.ConfigleamDashboardEndpoints.DeleteAccessKeyHandler))

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

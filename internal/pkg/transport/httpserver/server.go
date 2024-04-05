package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

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
	notify        NotifSet

	permissions PermissionsBuilder
}

func NewHttpServer(configuration ConfigurationSet, secrets SecretsSet, access AccessSet, dashboard DashboardSet, notify NotifSet, permissions PermissionsBuilder) *httpServer {
	return &httpServer{
		configuration: configuration,
		secrets:       secrets,
		access:        access,
		dashboard:     dashboard,
		permissions:   permissions,
		notify:        notify,
	}
}

func (s *httpServer) ListenAndServe(httpAddr string, enableTls bool) error {
	mux := http.NewServeMux()

	endpoints := newHandlers(s.configuration)

	// register health and readiness handlers
	mux.HandleFunc("/health", endpoints.HealthCheckHandler)
	mux.HandleFunc("/ready", endpoints.ReadinessCheckHandler)

	// middlewares
	auth := auth.NewAuthMiddleware(s.access, s.configuration, s.permissions, templates.New())

	// configuration business handlers
	mux.HandleFunc("GET /config", auth.Guard(p.ReadConfig)(s.configuration.ReadConfigHandler))

	// configuration clone environment business handlers
	mux.HandleFunc("POST /config/clone", auth.Guard(p.CloneEnvironment)(s.configuration.CloneConfigHandler))
	mux.HandleFunc("DELETE /config/clone", auth.Guard(p.CloneEnvironment)(s.configuration.DeleteConfigHandler))

	// configuration notify business handlers
	mux.HandleFunc("GET /config/sse", s.notify.NotifyHandler)

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
	staticDir := http.Dir("static")
	staticHandler := http.StripPrefix("/static/", http.FileServer(staticDir))
	mux.Handle("/static/", staticHandler)

	return s.startServer(httpAddr, mux, enableTls)
}

func (t *httpServer) Shutdown(ctx context.Context) error {
	if t.server != nil {
		return t.server.Shutdown(ctx)
	}
	return nil
}

func (s *httpServer) startServer(httpAddr string, mux http.Handler, enableTls bool) error {
	if enableTls {
		certPath := filepath.Join("certs", "cert.pem")
		keyPath := filepath.Join("certs", "key.pem")

		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			log.Println("failed to load TLS certificate:", err)
			return fmt.Errorf("failed to load TLS certificate: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		s.server = &http.Server{Addr: fmt.Sprintf(":%s", httpAddr), Handler: mux, TLSConfig: tlsConfig}

		log.Printf("Starting HTTPS server on port %s\n", httpAddr)
		return s.server.ListenAndServeTLS(certPath, keyPath)
	} else {
		s.server = &http.Server{Addr: fmt.Sprintf(":%s", httpAddr), Handler: mux}

		log.Printf("Starting HTTP server on port %s\n", httpAddr)
		return s.server.ListenAndServe()
	}
}

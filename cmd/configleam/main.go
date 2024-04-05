package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/access"
	"github.com/raw-leak/configleam/internal/app/configuration"
	"github.com/raw-leak/configleam/internal/app/dashboard"
	"github.com/raw-leak/configleam/internal/app/notify"
	"github.com/raw-leak/configleam/internal/app/secrets"
	"github.com/raw-leak/configleam/internal/pkg/encryptor"
	"github.com/raw-leak/configleam/internal/pkg/leaderelection"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	"github.com/raw-leak/configleam/internal/pkg/transport/httpserver"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err.Error())
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Get()
	if err != nil {
		return err
	}

	perms := permissions.New()
	encryptor, err := encryptor.NewEncryptor("")
	if err != nil {
		return err
	}

	accessSet, err := access.Init(ctx, cfg, encryptor, perms)
	if err != nil {
		return err
	}

	secretsSet, err := secrets.Init(ctx, cfg, encryptor)
	if err != nil {
		return err
	}

	notifySet, err := notify.Init(ctx, cfg)
	if err != nil {
		return err
	}

	configurationSet, err := configuration.Init(ctx, cfg, secretsSet, notifySet)
	if err != nil {
		return err
	}

	dashboardSet, err := dashboard.Init(ctx, cfg, accessSet.AccessService, configurationSet.ConfigurationService)
	if err != nil {
		return err
	}

	notifySet.RunLocal(ctx)

	if bool(cfg.EnableLeaderElection) {
		log.Println("Running with leader election")

		leConfig := leaderelection.LeaderElectionConfig{
			LeaseLockName:      cfg.LeaseLockName,
			LeaseLockNamespace: cfg.LeaseLockNamespace,
			Identity:           cfg.Hostname,
			LeaseDuration:      cfg.LeaseDuration,
			RenewDeadline:      cfg.RenewDeadline,
			RetryPeriod:        cfg.RetryPeriod,
		}

		elector, err := leaderelection.New(&leConfig, func() {
			log.Println("Started leading, starting service...")
			notifySet.RunGlobal(ctx)
			configurationSet.Run(ctx)
		}, func() {
			log.Println("Stopped leading, shutting down service...")
			notifySet.ShutdownGlobal()
			configurationSet.Shutdown()
		})
		if err != nil {
			return err
		}

		go elector.Run(ctx)
	} else {
		log.Println("Running without leader election")
		configurationSet.Run(ctx)
	}

	httpServer := httpserver.NewHttpServer(configurationSet, secretsSet, accessSet, dashboardSet, notifySet, perms)

	errChan := make(chan error, 2)
	go func(tls bool) {
		if err := httpServer.ListenAndServe(cfg.Port, tls); err != nil && err != http.ErrServerClosed {
			log.Println(err)
			errChan <- err
		}
	}(bool(cfg.Tls))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("Shutdown signal received")
	case <-errChan:
		log.Println("Received error from http server")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	if !bool(cfg.EnableLeaderElection) {
		configurationSet.Shutdown()
	}

	notifySet.ShutdownLocal(ctx)

	err = httpServer.Shutdown(ctx)
	if err != nil {
		log.Fatal("HTTP server shutdown error:", err)
	}

	log.Println("HTTP server gracefully shutdown")

	return nil
}

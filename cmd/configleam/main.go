package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam"
	"github.com/raw-leak/configleam/internal/pkg/leaderelection"
	"github.com/raw-leak/configleam/internal/pkg/transport/httpserver"
)

// TODOs:
// 2. Dynamic WS notif to the consumers
// 3. Secrets

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

	configleamSet, err := configleam.Init(ctx, cfg)
	if err != nil {
		return err
	}

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
			configleamSet.Service.Run(ctx)
		}, func() {
			log.Println("Stopped leading, shutting down service...")
			configleamSet.Service.Shutdown()
		})
		if err != nil {
			return err
		}

		go elector.Run(ctx)
	} else {
		log.Println("Running without leader election")
		configleamSet.Service.Run(ctx)
	}

	httpServer := httpserver.NewHttpTransport(configleamSet)

	errChan := make(chan error, 2)
	go func() {
		if err := httpServer.ListenAndServe(cfg.Port); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

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
		configleamSet.Service.Shutdown()
	}

	err = httpServer.Shutdown(ctx)
	if err != nil {
		log.Fatal("HTTP server shutdown error:", err)
	}

	log.Println("HTTP server gracefully shutdown")

	return nil
}

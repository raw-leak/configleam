package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam/analyzer"
	"github.com/raw-leak/configleam/internal/app/configleam/extractor"
	"github.com/raw-leak/configleam/internal/app/configleam/parser"
	"github.com/raw-leak/configleam/internal/app/configleam/repository"
	"github.com/raw-leak/configleam/internal/app/configleam/service"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/raw-leak/configleam/internal/transport/httpserver"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err.Error())
	}
}

func run() error {
	ctx := context.Background()
	cfg := config.Get()

	rdscli, err := rds.New(ctx, rds.RedisConfig{
		Addr:     cfg.RedisAddrs,
		Password: cfg.RedisPassword,
	})
	if err != nil {
		return err
	}

	rdsrepo := repository.NewRedisConfigRepository(rdscli)
	prsr := parser.New()
	exct := extractor.New()
	anlz := analyzer.New()

	// TODO: parameters
	s := service.New(service.ConfigleamServiceConfig{
		Envs:  []string{"develop", "release", "main"},
		Repos: []service.ConfigleamRepo{{Url: cfg.Url, Token: cfg.Token}},
	}, prsr, exct, rdsrepo, anlz)
	defer s.Shutdown()

	s.Run(ctx)

	httpServer := httpserver.NewHttpTransport()

	errChan := make(chan error, 2)

	go func() {

		if err := httpServer.ListenAndServe(cfg.Port); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("Shutdown signal received")
	case err := <-errChan:
		log.Printf("Server error: %v", err)
	}

	err = httpServer.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("HTTP server gracefully shutdown")

	return nil
}

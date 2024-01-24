package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam/service"
	"github.com/raw-leak/configleam/internal/transport/httpserver"
)

func main() {
	cfg := config.Get()

	s := service.NewConfigleamService(service.ConfigleamServiceConfig{
		Branches: []string{"develop"},
		Repos:    []service.ConfigleamRepo{{Url: cfg.Url, Token: cfg.Token}},
	})
	defer s.Shutdown()

	s.Run()

	httpServer := httpserver.NewHttpTransport()
	// grpcServer := grpcserver.NewGrpcTransport()

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

	err := httpServer.Shutdown(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("HTTP server gracefully shutdown")
}

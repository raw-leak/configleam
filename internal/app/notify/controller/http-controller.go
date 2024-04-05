package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/raw-leak/configleam/internal/app/notify/service"
)

type NotifyService interface {
	Subscribe(context.Context, string) *service.Client
	Unsubscribe(context.Context, *service.Client)
}

type NotifyEndpoints struct {
	service NotifyService
}

func New(s NotifyService) *NotifyEndpoints {
	return &NotifyEndpoints{s}
}

func (e NotifyEndpoints) NotifyHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		log.Println("ERROR: Streaming unsupported by client")
		return
	}

	env := r.URL.Query().Get("env")
	if env == "" {
		http.Error(w, "Environment not specified", http.StatusBadRequest)
		log.Println("ERROR: Request missing 'env' parameter")
		return
	}

	ctx := r.Context()
	clientIP := r.RemoteAddr
	log.Printf("SSE connection established with client %s for environment '%s'", clientIP, env)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	client := e.service.Subscribe(ctx, env)

	_, err := fmt.Fprintf(w, "data: %s\n\n", "Connection established")
	if err != nil {
		log.Printf("ERROR: Failed to write initial connection confirmation to %s: %v", clientIP, err)
		return
	}
	flusher.Flush()

	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				log.Printf("INFO: Channel closed for client %s, environment '%s'", clientIP, env)
				e.service.Unsubscribe(ctx, client)
				return
			}

			log.Printf("INFO: Sending update to client %s, environment '%s'", clientIP, env)
			_, err := fmt.Fprint(w, msg.Env)
			if err != nil {
				log.Printf("ERROR: Failed to send update to %s: %v", clientIP, err)
				e.service.Unsubscribe(ctx, client)
				return
			}

			flusher.Flush()
		case <-ctx.Done():
			log.Printf("INFO: Connection with client %s closed", clientIP)
			e.service.Unsubscribe(ctx, client)
			return
		}
	}
}

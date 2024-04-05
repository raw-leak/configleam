package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/raw-leak/configleam/internal/app/notify/repository"
)

type NotifyService struct {
	global     bool // Indicates if it runs with multiple instances or a single one.
	broker     *Broker
	repository repository.Repository
}

// New creates a new instance of the NotifyService.
func New(remote bool) *NotifyService {
	return &NotifyService{
		global: false,
		broker: NewBroker(),
	}
}

// RunLocal starts the local notification service, particularly the broker's run loop.
func (n *NotifyService) RunLocal(ctx context.Context) {
	go n.broker.Run(ctx)
}

// RunGlobal starts the global notification service
func (n *NotifyService) RunGlobal(ctx context.Context) {
	n.global = true
	n.repository.Subscribe(ctx, func(payload string) {
		var cu ConfigUpdate
		err := json.Unmarshal([]byte(payload), &cu)
		if err != nil {
			log.Printf("error unmarshaling received config update payload '%s': %v", payload, err)
			return
		}

		n.NotifyLocally(&cu)
	})
}

func (n *NotifyService) ShutdownGlobal() {
	n.global = false
	n.repository.Unsubscribe()
}

// NotifyConfigUpdate notifies all nodes and local subscribers about a configuration update.
func (n *NotifyService) NotifyConfigUpdate(ctx context.Context, repo, env, version string) {
	cu := &ConfigUpdate{Env: env, Repo: repo, Version: version}

	if n.global {
		if err := n.NotifyGlobally(ctx, cu); err != nil {
			log.Printf("Global notification error: %v", err)
			// TODO: Consider a retry mechanism or alternative action.
		}
	}

	n.NotifyLocally(cu)
}

func (n *NotifyService) NotifyGlobally(ctx context.Context, cu *ConfigUpdate) error {
	// Implementation for global notifications (e.g., via Redis pub/sub or etcd watchers).
	// Placeholder for actual implementation.

	jsonData, err := json.Marshal(cu)
	if err != nil {
		return fmt.Errorf("error marshaling config update: %v", err)
	}

	err = n.repository.Publish(ctx, string(jsonData))
	if err != nil {
		return fmt.Errorf("error publishing global notification: %v", err)
	}

	return nil
}

func (n *NotifyService) NotifyLocally(cu *ConfigUpdate) {
	n.broker.Broadcast(cu)
}

func (n *NotifyService) Subscribe(ctx context.Context, env string) *Client {
	client := &Client{
		Send: make(chan *ConfigUpdate),
		env:  env,
	}
	n.broker.Subscribe(ctx, client)
	return client
}

func (n *NotifyService) Unsubscribe(ctx context.Context, client *Client) {
	n.broker.Unsubscribe(ctx, client)
}

func (n *NotifyService) ShutdownLocal(ctx context.Context) {
	n.broker.Cleanup(ctx)
}

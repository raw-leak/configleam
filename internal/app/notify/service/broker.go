package service

import (
	"context"
)

// Broker manages clients and messages.
type Broker struct {
	clients     map[*Client]bool
	broadcast   chan *ConfigUpdate
	subscribe   chan *Client
	unsubscribe chan *Client
	shutdown    chan struct{}
	enabled     bool
}

// Client represents a subscriber with a channel to send messages.
type Client struct {
	Send chan *ConfigUpdate
	env  string
}

type ConfigUpdate struct {
	Env     string `json:"env"`
	Repo    string `json:"repo"`
	Version string `json:"version"`
}

func NewBroker() *Broker {
	return &Broker{
		broadcast:   make(chan *ConfigUpdate),
		subscribe:   make(chan *Client),
		unsubscribe: make(chan *Client),
		clients:     make(map[*Client]bool),
		shutdown:    make(chan struct{}),
		enabled:     true,
	}
}

func (b *Broker) Broadcast(data *ConfigUpdate) {
	if b.enabled {
		b.broadcast <- data
	}
}

func (b *Broker) Subscribe(ctx context.Context, client *Client) {
	if b.enabled {
		b.subscribe <- client
	}
}

func (b *Broker) Unsubscribe(ctx context.Context, client *Client) {
	if b.enabled {
		b.unsubscribe <- client
	}

}

func (b *Broker) Run(ctx context.Context) {
	for {
		select {
		case c, ok := <-b.subscribe:
			if ok {
				b.clients[c] = true
			}

		case client := <-b.unsubscribe:
			if _, ok := b.clients[client]; ok {
				close(client.Send)
				delete(b.clients, client)
			}

		case msg := <-b.broadcast:
			for client := range b.clients {
				if client.env == msg.Env {
					select {
					case client.Send <- msg:
					default:
						// Unable to send message, remove client
						// b.unsubscribe <- client
						// delete(b.clients, client)
						// close(client.Send)
					}
				}
			}

		case <-b.shutdown:
			for client := range b.clients {
				delete(b.clients, client)
				close(client.Send)
			}
			return
		}
	}
}

func (b *Broker) Cleanup(_ context.Context) {
	b.enabled = false
	close(b.shutdown)
}

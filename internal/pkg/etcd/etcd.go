package etcd

import (
	"context"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config struct {
}

type Etcd struct {
	Client *clientv3.Client
}

func New(ctx context.Context, config Config) (*Etcd, error) {
	return nil, nil
	client, err := clientv3.New(clientv3.Config{})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = client.Status(ctx, client.Endpoints()[0])
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return &Etcd{Client: client}, nil
}

func (e *Etcd) Disconnect(ctx context.Context) {
	if e.Client != nil {
		if err := e.Client.Close(); err != nil {
			log.Printf("Failed to close Etcd client: %v", err)
		}
	}
}

package etcd

import (
	"context"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdConfig struct {
	EtcdAddrs    []string
	EtcdUsername string
	EtcdPassword string
}

type Etcd struct {
	Client *clientv3.Client
}

func New(ctx context.Context, config EtcdConfig) (*Etcd, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: config.EtcdAddrs,
		Username:  config.EtcdUsername,
		Password:  config.EtcdPassword,
	})
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
	return &Etcd{client}, nil
}

func (e *Etcd) Disconnect(ctx context.Context) {
	if e.Client != nil {
		if err := e.Client.Close(); err != nil {
			log.Printf("Failed to close Etcd client: %v", err)
		}
	}
}

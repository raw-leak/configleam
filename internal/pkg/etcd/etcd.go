package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"path/filepath"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdConfig struct {
	EtcdAddrs    []string
	EtcdUsername string
	EtcdPassword string
	TLS          bool
}

type Etcd struct {
	Client *clientv3.Client
}

func New(ctx context.Context, config EtcdConfig) (*Etcd, error) {
	var tlsConfig *tls.Config

	if config.TLS {
		certPath := filepath.Join("certs", "etcd-cert.pem")
		keyPath := filepath.Join("certs", "etcd-key.pem")

		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			log.Println("failed to load TLS certificate for etcd:", err)
			return nil, fmt.Errorf("failed to load TLS certificate for etcd: %v", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints: config.EtcdAddrs,
		Username:  config.EtcdUsername,
		Password:  config.EtcdPassword,
		TLS:       tlsConfig,
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

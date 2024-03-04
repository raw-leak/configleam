package rds

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	Username string
	DB       int
	TLS      bool
}

type Redis struct {
	Client *redis.Client
}

// New creates a new Redis client with the provided configuration
func New(ctx context.Context, config RedisConfig) (*Redis, error) {
	var tlsConfig *tls.Config

	if config.TLS {
		certPath := filepath.Join("certs", "redis-cert.pem")
		keyPath := filepath.Join("certs", "redis-key.pem")

		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			log.Println("failed to load TLS certificate for Redis:", err)
			return nil, fmt.Errorf("failed to load TLS certificate for Redis: %v", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:      config.Addr,
		Password:  config.Password,
		DB:        config.DB,
		Username:  config.Username,
		TLSConfig: tlsConfig,
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Printf("Redis client connected successfully to %s\n", config.Addr)

	return &Redis{Client: rdb}, nil
}

// Disconnect handles the disconnection logic for Redis client
func (r *Redis) Disconnect(ctx context.Context) {
	if r.Client != nil {
		if err := r.Client.Close(); err != nil {
			log.Printf("Failed to close Redis client: %v", err)
		}
	}
}

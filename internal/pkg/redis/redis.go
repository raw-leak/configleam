package rds

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type Redis struct {
	Client *redis.Client
}

// New creates a new Redis client with the provided configuration
func New(ctx context.Context, config RedisConfig) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password, // no password set by default
		DB:       config.DB,       // use default DB
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

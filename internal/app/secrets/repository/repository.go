package repository

import (
	"context"
	"fmt"

	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

type Repository interface {
	GetSecret(ctx context.Context, env string, key string) (interface{}, error)
	UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error
	CloneSecrets(ctx context.Context, cloneEnv, newEnv string) error
	HealthCheck(ctx context.Context) error
}

type RepositoryConfig struct {
	RedisAddrs    string
	RedisUsername string
	RedisPassword string
	RedisTLS      bool

	EtcdAddrs    string
	EtcdUsername string
	EtcdPassword string
}

func New(ctx context.Context, cfg RepositoryConfig, encryptor Encryptor) (Repository, error) {
	if cfg.RedisAddrs == "" {
		return nil, fmt.Errorf("RedisAddress is no provided")
	}

	redisCli, err := rds.New(ctx, rds.RedisConfig{
		Addr:     cfg.RedisAddrs,
		Password: cfg.RedisPassword,
		Username: cfg.RedisUsername,
		TLS:      cfg.RedisTLS,
	})
	if err != nil {
		return nil, err
	}

	return NewRedisRepository(redisCli, encryptor), nil

	// TODO: add support for ETCD
	// if cfg.EtcdAddrs == "" {
	// 	return nil, fmt.Errorf("EtcdAddrs is no provided")
	// }

	// etcdCli, err := etcd.New(ctx, etcd.EtcdConfig{
	// 	EtcdAddrs:    cfg.EtcdAddrs,
	// 	EtcdUsername: cfg.EtcdUsername,
	// 	EtcdPassword: cfg.EtcdPassword,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// return NewEtcdRepository(etcdCli), nil
}

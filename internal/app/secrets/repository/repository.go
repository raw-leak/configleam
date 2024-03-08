package repository

import (
	"context"
	"fmt"

	"github.com/raw-leak/configleam/internal/pkg/etcd"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

const (
	SecretPrefix = "configleam:secret"
)

type Encryptor interface {
	Encrypt(ctx context.Context, b []byte) ([]byte, error)
	Decrypt(ctx context.Context, b []byte) ([]byte, error)
}

type Repository interface {
	GetSecret(ctx context.Context, env string, key string) (interface{}, error)
	UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error
	CloneSecrets(ctx context.Context, cloneEnv, newEnv string) error
}

type RepositoryConfig struct {
	RedisAddrs    string
	RedisUsername string
	RedisPassword string
	RedisTLS      bool

	EtcdAddrs    []string
	EtcdUsername string
	EtcdPassword string
	EtcdTLS      bool
}

func New(ctx context.Context, cfg RepositoryConfig, encryptor Encryptor) (Repository, error) {
	if cfg.RedisAddrs != "" {
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
	}

	if len(cfg.EtcdAddrs) > 0 {
		etcdCli, err := etcd.New(ctx, etcd.EtcdConfig{
			EtcdAddrs:    cfg.EtcdAddrs,
			EtcdUsername: cfg.EtcdUsername,
			EtcdPassword: cfg.EtcdPassword,
			TLS:          cfg.EtcdTLS,
		})
		if err != nil {
			return nil, err
		}
		return NewEtcdRepository(etcdCli, encryptor), nil
	}

	return nil, fmt.Errorf("'RedisAddress' nor 'EtcdAddrs' has been provided for 'secrets' repository")
}

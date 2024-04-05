package repository

import (
	"context"
	"fmt"

	"github.com/raw-leak/configleam/internal/pkg/etcd"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

const (
	NotifyPrefix = "configleam:notify"
)

type Repository interface {
	Publish(ctx context.Context, payload string) error
	Subscribe(ctx context.Context, callback func(string))
	Unsubscribe()
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

func New(ctx context.Context, cfg RepositoryConfig) (Repository, error) {
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

		return NewRedisRepository(redisCli), nil
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
		return NewEtcdRepository(etcdCli), nil
	}

	return nil, fmt.Errorf("'RedisAddress' nor 'EtcdAddrs' has been provided for 'notify' repository")
}

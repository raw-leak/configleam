package repository

import (
	"context"
	"fmt"

	"github.com/raw-leak/configleam/internal/app/configleam-access/service"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

type RepositoryConfig struct {
	RedisAddrs    string
	RedisUsername string
	RedisPassword string

	EtcdAddrs    string
	EtcdUsername string
	EtcdPassword string
}

func New(ctx context.Context, cfg RepositoryConfig, encryptor Encryptor) (service.Repository, error) {
	if cfg.RedisAddrs == "" {
		return nil, fmt.Errorf("RedisAddress is no provided")
	}

	redisCli, err := rds.New(ctx, rds.RedisConfig{
		Addr:     cfg.RedisAddrs,
		Password: cfg.RedisPassword,
		Username: cfg.RedisUsername,
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

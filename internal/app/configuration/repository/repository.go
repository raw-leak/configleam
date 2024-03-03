package repository

import (
	"context"
	"fmt"

	"github.com/raw-leak/configleam/internal/app/configuration/types"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

type Repository interface {
	CloneConfig(ctx context.Context, env, newEnv string, updateGlobals map[string]interface{}) error
	ReadConfig(ctx context.Context, env string, groups, globalKeys []string) (map[string]interface{}, error)
	UpsertConfig(ctx context.Context, env string, gitRepoName string, config *types.ParsedRepoConfig) error
	DeleteConfig(ctx context.Context, env string, gitRepoName string) error

	AddEnv(ctx context.Context, envName string, params EnvParams) error
	DeleteEnv(ctx context.Context, envName string) error
	GetEnvOriginal(ctx context.Context, envName string) (string, bool, error)
	SetEnvVersion(ctx context.Context, envName string, version string) error
	GetAllEnvs(ctx context.Context) ([]EnvParams, error)
	GetEnvParams(ctx context.Context, envName string) (EnvParams, error)

	HealthCheck(ctx context.Context) error
}

type RepositoryConfig struct {
	RedisAddrs    string
	RedisUsername string
	RedisPassword string

	EtcdAddrs    string
	EtcdUsername string
	EtcdPassword string
}

func New(ctx context.Context, cfg RepositoryConfig) (Repository, error) {
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

	return NewRedisRepository(redisCli), nil

	// TODO: add support to ETCD
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

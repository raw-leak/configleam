package repository

import (
	"context"
	"fmt"

	"github.com/raw-leak/configleam/internal/app/configuration/types"
	"github.com/raw-leak/configleam/internal/pkg/etcd"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

const (
	ConfigurationPrefix     = "configleam:config"
	ConfigurationEnvPrefix  = "configleam:env"
	ConfigurationLockPrefix = "configleam:lock"

	GlobalPrefix = "global"
	GroupPrefix  = "group"
)

type Repository interface {
	CloneConfig(ctx context.Context, repo, env, newEnv string, updateGlobals map[string]interface{}) error
	ReadConfig(ctx context.Context, repo, env string, groups, globalKeys []string) (map[string]interface{}, error)
	UpsertConfig(ctx context.Context, repo, env string, config *types.ParsedRepoConfig) error
	DeleteConfig(ctx context.Context, repo, env string) error

	AddEnv(ctx context.Context, env string, params EnvParams) error
	DeleteEnv(ctx context.Context, env string) error
	GetEnvOriginal(ctx context.Context, env string) (string, bool, error)
	SetEnvVersion(ctx context.Context, env string, v string) error
	GetAllEnvs(ctx context.Context) ([]EnvParams, error)
	GetEnvParams(ctx context.Context, env string) (EnvParams, error)
}

type RepositoryConfig struct {
	RedisAddrs    string
	RedisUsername string
	RedisPassword string

	EtcdAddrs    []string
	EtcdUsername string
	EtcdPassword string
}

func New(ctx context.Context, cfg RepositoryConfig) (Repository, error) {
	if cfg.RedisAddrs != "" {
		redisCli, err := rds.New(ctx, rds.RedisConfig{
			Addr:     cfg.RedisAddrs,
			Password: cfg.RedisPassword,
			Username: cfg.RedisUsername,
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
		})
		if err != nil {
			return nil, err
		}

		return NewEtcdRepository(etcdCli), nil
	}

	return nil, fmt.Errorf("'RedisAddress' nor 'EtcdAddrs' has been provided for 'configuration' repository")
}

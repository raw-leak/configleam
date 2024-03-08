package repository

import (
	"context"
	"fmt"

	"github.com/raw-leak/configleam/internal/pkg/etcd"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

const (
	AccessPrefix = "configleam:access"
	KeyPrefix    = "key"
	MetaPrefix   = "meta"
	SetPrefix    = "set"
)

type Repository interface {
	StoreAccessKey(ctx context.Context, accessKey AccessKey) error
	GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error)
	PaginateAccessKeys(ctx context.Context, page int, size int) (*PaginatedAccessKeys, error)
	RemoveKeys(ctx context.Context, keys []string) error
}

type Encryptor interface {
	Encrypt(ctx context.Context, b []byte) ([]byte, error)
	EncryptDet(ctx context.Context, b []byte) ([]byte, error)
	Decrypt(ctx context.Context, b []byte) ([]byte, error)
}

type RepositoryConfig struct {
	RedisAddrs    string
	RedisUsername string
	RedisPassword string

	EtcdAddrs    []string
	EtcdUsername string
	EtcdPassword string
}

func New(ctx context.Context, cfg RepositoryConfig, encryptor Encryptor) (Repository, error) {
	if cfg.RedisAddrs != "" {
		redisCli, err := rds.New(ctx, rds.RedisConfig{
			Addr:     cfg.RedisAddrs,
			Password: cfg.RedisPassword,
			Username: cfg.RedisUsername,
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
		})
		if err != nil {
			return nil, err
		}

		return NewEtcdRepository(etcdCli, encryptor), nil
	}

	return nil, fmt.Errorf("'RedisAddress' nor 'EtcdAddrs' has been provided for 'access' repository")
}

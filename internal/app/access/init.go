package access

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/access/controller"
	"github.com/raw-leak/configleam/internal/app/access/interfaces"
	"github.com/raw-leak/configleam/internal/app/access/keys"
	"github.com/raw-leak/configleam/internal/app/access/repository"
	"github.com/raw-leak/configleam/internal/app/access/service"
)

type AccessSet struct {
	*service.AccessService
	*controller.AccessEndpoints
}

func Init(ctx context.Context, cfg *config.Config, encryptor repository.Encryptor, perms interfaces.Permissions) (*AccessSet, error) {
	repo, err := repository.New(ctx, repository.RepositoryConfig{
		RedisAddrs:    cfg.RedisAddrs,
		RedisUsername: cfg.RedisUsername,
		RedisPassword: cfg.RedisPassword,
		RedisTLS:      bool(cfg.RedisTls),

		EtcdAddrs:    cfg.EtcdAddrs,
		EtcdUsername: cfg.EtcdUsername,
		EtcdPassword: cfg.EtcdPassword,
	}, encryptor)
	if err != nil {
		return nil, err

	}

	keys := keys.New()

	service := service.New(keys, perms, repo)

	endpoints := controller.New(service)

	return &AccessSet{
		service,
		endpoints,
	}, nil
}

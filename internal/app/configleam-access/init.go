package configleamsecrets

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam-access/controller"
	"github.com/raw-leak/configleam/internal/app/configleam-access/keys"
	"github.com/raw-leak/configleam/internal/app/configleam-access/repository"
	"github.com/raw-leak/configleam/internal/app/configleam-access/service"
)

type ConfigleamAccessSet struct {
	*service.ConfigleamAccess
	*controller.ConfigleamAccessEndpoints
}

func Init(ctx context.Context, cfg *config.Config, encryptor repository.Encryptor, perms service.Permissions) (*ConfigleamAccessSet, error) {
	repo, err := repository.New(ctx, repository.RepositoryConfig{
		RedisAddrs:    cfg.RedisAddrs,
		RedisUsername: cfg.RedisUsername,
		RedisPassword: cfg.RedisPassword,

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

	return &ConfigleamAccessSet{
		service,
		endpoints,
	}, nil
}

package configleamsecrets

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam-secrets/controller"
	"github.com/raw-leak/configleam/internal/app/configleam-secrets/repository"
	"github.com/raw-leak/configleam/internal/app/configleam-secrets/service"
)

type ConfigleamSecretsSet struct {
	*service.ConfigleamSecrets
	*controller.ConfigleamSecretsEndpoints
}

func Init(ctx context.Context, cfg *config.Config, encryptor repository.Encryptor) (*ConfigleamSecretsSet, error) {
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

	service := service.New(repo)

	endpoints := controller.New(service)

	return &ConfigleamSecretsSet{
		service, endpoints,
	}, nil
}

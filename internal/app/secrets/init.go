package secrets

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/secrets/controller"
	"github.com/raw-leak/configleam/internal/app/secrets/repository"
	"github.com/raw-leak/configleam/internal/app/secrets/service"
)

type SecretsSet struct {
	*service.SecretsService
	*controller.SecretsEndpoints
}

func Init(ctx context.Context, cfg *config.Config, encryptor repository.Encryptor) (*SecretsSet, error) {
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

	return &SecretsSet{
		service, endpoints,
	}, nil
}

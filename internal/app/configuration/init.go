package configuration

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configuration/analyzer"
	"github.com/raw-leak/configleam/internal/app/configuration/controller"
	"github.com/raw-leak/configleam/internal/app/configuration/extractor"
	"github.com/raw-leak/configleam/internal/app/configuration/parser"
	"github.com/raw-leak/configleam/internal/app/configuration/repository"
	"github.com/raw-leak/configleam/internal/app/configuration/service"
)

type ConfigurationSet struct {
	*service.ConfigurationService
	*controller.ConfigurationEndpoints
}

func Init(ctx context.Context, cfg *config.Config, secrets service.Secrets) (*ConfigurationSet, error) {
	repo, err := repository.New(ctx, repository.RepositoryConfig{
		RedisAddrs:    cfg.RedisAddrs,
		RedisUsername: cfg.RedisUsername,
		RedisPassword: cfg.RedisPassword,
		RedisTLS:      bool(cfg.RedisTls),

		EtcdAddrs:    cfg.EtcdAddrs,
		EtcdUsername: cfg.EtcdUsername,
		EtcdPassword: cfg.EtcdPassword,
	})
	if err != nil {
		return nil, err
	}

	parser := parser.New()
	extractor := extractor.New()
	analyzer := analyzer.New()

	service := service.New(service.ConfigurationConfig{
		Branch:       cfg.RepoBranch,
		RepoUrl:      cfg.RepoUrl,
		Envs:         cfg.RepoEnvs,
		PullInterval: cfg.PullInterval,
	}, parser, extractor, repo, analyzer, secrets)

	endpoints := controller.New(service)

	return &ConfigurationSet{
		service, endpoints,
	}, nil
}

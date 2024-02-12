package configleam

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam/analyzer"
	"github.com/raw-leak/configleam/internal/app/configleam/controller"
	"github.com/raw-leak/configleam/internal/app/configleam/extractor"
	"github.com/raw-leak/configleam/internal/app/configleam/parser"
	"github.com/raw-leak/configleam/internal/app/configleam/repository"
	"github.com/raw-leak/configleam/internal/app/configleam/service"
)

type ConfigleamSet struct {
	*service.ConfigleamService
	*controller.ConfigleamEndpoints
}

func Init(ctx context.Context, cfg *config.Config, secrets service.Secrets) (*ConfigleamSet, error) {
	repo, err := repository.New(ctx, repository.RepositoryConfig{
		RedisAddrs:    cfg.RedisAddrs,
		RedisUsername: cfg.RedisUsername,
		RedisPassword: cfg.RedisPassword,

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

	service := service.New(service.ConfigleamServiceConfig{
		Branch:       cfg.RepoConfig.Branch,
		Envs:         cfg.RepoConfig.Environments,
		Repos:        cfg.RepoConfig.Repositories,
		PullInterval: cfg.PullInterval,
	}, parser, extractor, repo, analyzer, secrets)

	endpoints := controller.New(service)

	return &ConfigleamSet{
		service, endpoints,
	}, nil
}

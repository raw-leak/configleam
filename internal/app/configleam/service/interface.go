package service

import (
	"context"

	"github.com/raw-leak/configleam/internal/app/configleam/analyzer"
	"github.com/raw-leak/configleam/internal/app/configleam/gitmanager"
	"github.com/raw-leak/configleam/internal/app/configleam/types"
)

type Extractor interface {
	ExtractConfigList(dir string) (*types.ExtractedConfigList, error)
}

type Parser interface {
	ParseConfigList(*types.ExtractedConfigList) (*types.ParsedRepoConfig, error)
}

type Analyzer interface {
	AnalyzeTagsForUpdates(envs map[string]gitmanager.Env, tags []string) ([]analyzer.EnvUpdate, bool, error)
}

type Repository interface {
	ReadConfig(ctx context.Context, env string, groups, globalKeys []string) (map[string]interface{}, error)
	UpsertConfig(ctx context.Context, env string, gitRepoName string, config *types.ParsedRepoConfig) error
	HealthCheck(ctx context.Context) error
}

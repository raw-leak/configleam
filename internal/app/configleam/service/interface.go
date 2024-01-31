package service

import (
	"context"

	"github.com/raw-leak/configleam/internal/app/configleam/analyzer"
	"github.com/raw-leak/configleam/internal/app/configleam/gitmanager"
	"github.com/raw-leak/configleam/internal/app/configleam/types"
)

type Extractor interface {
	ExtractConfigList(string) (*types.ExtractedConfigList, error)
}

type Parser interface {
	ParseConfigList(*types.ExtractedConfigList) (*types.ParsedRepoConfig, error)
}

type Analyzer interface {
	AnalyzeTagsForUpdates(envs []gitmanager.Env, tags []string) ([]analyzer.EnvUpdate, bool, error)
}

type ConfigRepository interface {
	StoreConfig(ctx context.Context, config *types.ParsedRepoConfig) error
	ReadConfig(ctx context.Context, groups []string, globalKeys []string) (map[string]interface{}, error)
	DeleteEnvConfigs(ctx context.Context, envName, gitRepoName string) error
}

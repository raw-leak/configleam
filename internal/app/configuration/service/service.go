package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/raw-leak/configleam/internal/app/configuration/analyzer"
	"github.com/raw-leak/configleam/internal/app/configuration/gitmanager"
	"github.com/raw-leak/configleam/internal/app/configuration/repository"
	"github.com/raw-leak/configleam/internal/app/configuration/types"
	"github.com/raw-leak/configleam/internal/pkg/auth"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

const (
	PullIntervalDefault = 5 * time.Second
)

type Secrets interface {
	InsertSecrets(ctx context.Context, env string, cfg *map[string]interface{}, populate bool) error
	CloneSecrets(ctx context.Context, env, newEnv string) error
}

type Extractor interface {
	ExtractConfigList(dir string) (*types.ExtractedConfigList, error)
}

type Parser interface {
	ParseConfigList(*types.ExtractedConfigList) (*types.ParsedRepoConfig, error)
}

type Analyzer interface {
	AnalyzeTagsForUpdates(envs map[string]gitmanager.Env, tags []string) ([]analyzer.EnvUpdate, bool, error)
}

type ConfigurationService struct {
	gitrepo *gitmanager.GitRepository
	envs    map[string]bool

	mux          sync.RWMutex
	pollInterval time.Duration
	ticker       *time.Ticker

	repository repository.Repository
	extractor  Extractor
	parser     Parser
	analyzer   Analyzer
	secrets    Secrets
}

type ConfigurationConfig struct {
	RepoUrl      string
	Envs         []string
	Branch       string
	PullInterval time.Duration
}

func New(cfg ConfigurationConfig, parser Parser, extractor Extractor, repository repository.Repository, analyzer Analyzer, secrets Secrets) *ConfigurationService {
	gitrepo, err := gitmanager.NewGitRepository(cfg.RepoUrl, cfg.Branch, cfg.Envs)
	if err != nil {
		log.Fatalf("Fatal generating '%s' local git-repository", cfg.RepoUrl)
	}

	if cfg.PullInterval == 0 {
		cfg.PullInterval = PullIntervalDefault
	}

	envs := map[string]bool{}
	for _, env := range cfg.Envs {
		envs[env] = true
	}

	return &ConfigurationService{
		gitrepo:      gitrepo,
		envs:         envs,
		pollInterval: cfg.PullInterval,
		mux:          sync.RWMutex{},
		repository:   repository,
		extractor:    extractor,
		parser:       parser,
		analyzer:     analyzer,
		secrets:      secrets,
	}
}

func (s *ConfigurationService) Run(ctx context.Context) {
	err := s.cloneAllRemoteRepos(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = s.buildConfigFromLocalFirstTime(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	go s.watchRemoteReposForUpdates()
}

func (s *ConfigurationService) ReadConfig(ctx context.Context, env string, groups, globals []string) (map[string]interface{}, error) {
	if env == "" {
		return nil, errors.New("env cannot be empty")
	}

	accessKeyPerms, ok := ctx.Value(auth.AccessKeyContextKey{}).(permissions.AccessKeyPermissions)
	if !ok {
		return nil, errors.New("permissions were not found")
	}

	cfg, err := s.repository.ReadConfig(ctx, s.gitrepo.Name, env, groups, globals)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	err = s.secrets.InsertSecrets(ctx, env, &cfg, accessKeyPerms.CanRevealSecrets(env))
	if err != nil {
		return nil, fmt.Errorf("failed to insert secrets: %w", err)
	}

	return cfg, nil
}

func (s *ConfigurationService) cloneAllRemoteRepos(_ context.Context) error {
	err := s.gitrepo.CloneRemoteRepo()
	if err != nil {
		return err
	}

	return nil
}

func (s *ConfigurationService) buildConfigFromLocalFirstTime(ctx context.Context) error {
	err := s.buildConfigFromLocalRepo(ctx)
	if err != nil {
		log.Printf("Error while building the config from a local repo: %e\n", err)
		return err
	}

	addedEnvs := []string{}
	for _, env := range s.gitrepo.Envs {
		envParams := repository.EnvParams{
			Name:    env.Name,
			Version: env.LastTag,
			Clone:   false,
		}
		err = s.repository.AddEnv(ctx, env.Name, envParams)
		if err != nil {
			log.Printf("Error while adding environment '%s' to the repository: %e\n", s.gitrepo.Name, err)
			for _, addedEnv := range addedEnvs {
				delErr := s.repository.DeleteEnv(ctx, addedEnv)
				if delErr != nil {
					log.Printf("Error while deleting environment '%s' after failed adding environment: %e\n", addedEnv, err)
				}
			}
			return err
		}
		addedEnvs = append(addedEnvs, env.Name)
	}

	return nil
}

func (s *ConfigurationService) watchRemoteReposForUpdates() {
	s.ticker = time.NewTicker(s.pollInterval)

	for range s.ticker.C {
		err := s.buildConfigFromLocalRepo(context.Background())
		if err != nil {
			log.Printf("Error on watching while building the config from a local repo: %e\n", err)
		}
	}
}

func (s *ConfigurationService) buildConfigFromLocalRepo(ctx context.Context) error {
	tags, err := s.gitrepo.PullTagsFromRemoteRepo()
	if err != nil {
		log.Println("Error pulling tags:", err)
		return err
	}

	updatedEnvs, ok, err := s.analyzer.AnalyzeTagsForUpdates(s.gitrepo.Envs, tags)
	if err != nil {
		log.Println("Error analyzing tags:", err)
		return err

	}
	if !ok {
		log.Printf("There are no changes for repo [%s]", s.gitrepo.URL)
		return nil
	}

	for _, env := range updatedEnvs {
		log.Printf("New changes detected, applying updates for [%s] with the new tag [%s]", env.Name, env.Tag)

		s.gitrepo.FetchAndCheckout(env.Tag)

		// need to lock the repo from change while extracting the config-list
		s.gitrepo.Mux.Lock()
		configList, err := s.extractor.ExtractConfigList(s.gitrepo.Dir + "/" + env.Name)
		s.gitrepo.Mux.Unlock()

		if err != nil {
			log.Println("Error extracting configuration:", err)
			return err
		}

		repoConfig, err := s.parser.ParseConfigList(configList)
		if err != nil {
			log.Println("Error parsing configuration list:", err)
			return err
		}

		log.Printf("Upserting new repo config for environment '%s'", env.Name)
		err = s.repository.UpsertConfig(ctx, s.gitrepo.Name, env.Name, repoConfig)
		if err != nil {
			log.Printf("Error upserting config for environment '%s' with error %v:", env.Name, err)
			return err
		}

		s.gitrepo.SetEnvLatestVersion(ctx, env.Name, env.Tag, env.SemVer)
	}

	return nil
}

func (s *ConfigurationService) cleanLocalRepos() {
	log.Println("Cleaning local repositories...")

	err := s.gitrepo.RemoveLocalRepo()
	if err != nil {
		log.Printf("Error on removing local repo %s from dir %s", s.gitrepo.URL, s.gitrepo.Dir)
	}

	// TODO: do we need to clear the repo environments once down?
}

func (s *ConfigurationService) Shutdown() {
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	s.cleanLocalRepos()
}

func (s *ConfigurationService) DeleteConfig(ctx context.Context, deleteEnv string) error {
	for reservedEnv := range s.envs {
		if reservedEnv == deleteEnv {
			return fmt.Errorf("env '%s' reserved and can not be deleted", deleteEnv)
		}
	}

	err := s.repository.DeleteConfig(ctx, s.gitrepo.Name, deleteEnv)
	if err != nil {
		log.Printf("Error deleting config environment '%s' with error %v:", deleteEnv, err)
		return err
	}

	err = s.repository.DeleteEnv(ctx, deleteEnv)
	if err != nil {
		log.Printf("Error deleting environment '%s' with error %v:", deleteEnv, err)
		return err
	}

	return nil
}

func (s *ConfigurationService) CloneConfig(ctx context.Context, env, newEnv string, updateGlobals map[string]interface{}) error {
	found := false

	for e := range s.envs {
		if e == env {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("env %s for cloning has not been found", env)
	}

	s.gitrepo.Mux.Lock()
	defer s.gitrepo.Mux.Unlock()

	if err := s.repository.CloneConfig(ctx, s.gitrepo.Name, env, newEnv, updateGlobals); err != nil {
		return err
	}

	if err := s.secrets.CloneSecrets(ctx, env, newEnv); err != nil {
		log.Printf("Error cloning secrets for %s to %s: %v", env, newEnv, err)
		if delErr := s.repository.DeleteConfig(ctx, s.gitrepo.Name, newEnv); delErr != nil {
			log.Printf("Error cleaning up config for '%s' after failed secrets clone: %v", newEnv, delErr)
		}
		return err
	}

	newEnvParams := repository.EnvParams{
		Name:     s.gitrepo.Name,
		Version:  s.gitrepo.LastTag,
		Clone:    true,
		Original: env,
	}
	err := s.repository.AddEnv(ctx, newEnv, newEnvParams)
	if err != nil {
		log.Printf("Error adding clone '%s' of environment '%s': %v", env, newEnv, err)
		if delErr := s.repository.DeleteConfig(ctx, s.gitrepo.Name, newEnv); delErr != nil {
			log.Printf("Error cleaning up config for %s after failed adding clone: %v", newEnv, delErr)
		}
		return err
	}

	return nil
}

func (s *ConfigurationService) GetEnvs(ctx context.Context) []string {
	envs := make([]string, 0, len(s.envs))

	for env := range s.envs {
		envs = append(envs, env)
	}

	return envs
}

func (s *ConfigurationService) IsEnvOriginal(ctx context.Context, env string) bool {
	_, ok := s.envs[env]
	return ok
}

func (s *ConfigurationService) GetEnvOriginal(ctx context.Context, env string) (string, bool, error) {
	return s.repository.GetEnvOriginal(ctx, env)
}

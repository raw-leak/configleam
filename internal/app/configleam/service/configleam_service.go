package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/raw-leak/configleam/internal/app/configleam/gitmanager"
)

// TODO:
// 3. Configuration Management:
// The handling of configuration (like polling intervals) can potentially be abstracted out or managed
// in a more centralized way to allow easier adjustments and scalability.

type ConfigleamService struct {
	gitrepos []*gitmanager.GitRepository

	mux          sync.RWMutex
	pollInterval time.Duration
	ticker       *time.Ticker

	repository Repository
	extractor  Extractor
	parser     Parser
	analyzer   Analyzer
}

type ConfigleamServiceConfig struct {
	Repos  []string
	Envs   []string
	Branch string
}

func New(cfg ConfigleamServiceConfig, parser Parser, extractor Extractor, repository Repository, analyzer Analyzer) *ConfigleamService {
	gitrepos := []*gitmanager.GitRepository{}

	for _, url := range cfg.Repos {
		repo, err := gitmanager.NewGitRepository(url, cfg.Branch, cfg.Envs)
		if err != nil {
			log.Fatalf("Fatal generating '%s' local git-repository", url)
		}
		gitrepos = append(gitrepos, repo)
	}

	return &ConfigleamService{
		gitrepos:     gitrepos,
		pollInterval: 5 * time.Second,
		mux:          sync.RWMutex{},
		repository:   repository,
		extractor:    extractor,
		parser:       parser,
		analyzer:     analyzer,
	}
}

func (s *ConfigleamService) Run(ctx context.Context) {
	err := s.cloneAllRemoteRepos(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = s.buildConfigFromLocalRepos(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	go s.watchRemoteReposForUpdates()
}

func (s *ConfigleamService) ReadConfig(ctx context.Context, env string, groups, globals []string) (map[string]interface{}, error) {
	cfg, err := s.repository.ReadConfig(ctx, env, groups, globals)
	if err != nil {
		// TODO: log
		return nil, err
	}

	return cfg, nil
}

func (s *ConfigleamService) cloneAllRemoteRepos(_ context.Context) error {
	for _, gitrepo := range s.gitrepos {
		err := gitrepo.CloneRemoteRepo()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ConfigleamService) buildConfigFromLocalRepos(ctx context.Context) error {
	for _, gitrepo := range s.gitrepos {
		err := s.buildConfigFromLocalRepo(ctx, gitrepo)
		if err != nil {
			log.Printf("Error while building the config from a local repo: %e\n", err)
			return err
		}
	}

	return nil
}

func (s *ConfigleamService) watchRemoteReposForUpdates() {
	s.ticker = time.NewTicker(s.pollInterval)

	for range s.ticker.C {
		for _, gitrepo := range s.gitrepos {
			err := s.buildConfigFromLocalRepo(context.Background(), gitrepo)
			if err != nil {
				log.Printf("Error on watching while building the config from a local repo: %e\n", err)

			}
		}
	}
}

func (s *ConfigleamService) buildConfigFromLocalRepo(ctx context.Context, gitrepo *gitmanager.GitRepository) error {
	tags, err := gitrepo.PullTagsFromRemoteRepo()
	if err != nil {
		log.Println("Error pulling tags:", err)
		return err
	}

	updatedEnvs, ok, err := s.analyzer.AnalyzeTagsForUpdates(gitrepo.Envs, tags)
	if err != nil {
		log.Println("Error analyzing tags:", err)
		return err

	}
	if !ok {
		log.Printf("There are no changes for repo [%s]", gitrepo.URL)
		return nil
	}

	for _, env := range updatedEnvs {
		log.Printf("New changes detected, applying updates for [%s] with the new tag [%s]", env.Name, env.Tag)

		gitrepo.FetchAndCheckout(env.Tag)

		// need to lock the repo from change while extracting the config-list
		gitrepo.Mux.Lock()
		configList, err := s.extractor.ExtractConfigList(gitrepo.Dir + "/" + env.Name)
		gitrepo.Mux.Unlock()

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
		err = s.repository.UpsertConfig(ctx, env.Name, gitrepo.Name, repoConfig)
		if err != nil {
			log.Printf("Error upserting config for environment '%s' with error %v:", env.Name, err)
			return err
		}

		gitrepo.SetEnvLatestVersion(ctx, env.Name, env.Tag, env.SemVer)
	}

	return nil
}

func (s *ConfigleamService) cleanLocalRepos() {
	log.Println("Cleaning local repositories...")

	for _, gitrepo := range s.gitrepos {
		err := gitrepo.RemoveLocalRepo()
		if err != nil {
			log.Printf("Error on removing local repo %s from dir %s", gitrepo.URL, gitrepo.Dir)
		}
	}
}

func (s *ConfigleamService) Shutdown() {
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	s.cleanLocalRepos()
}

func (s *ConfigleamService) HealthCheck(ctx context.Context) error {
	return s.repository.HealthCheck(ctx)
}

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

type configleamService struct {
	gitrepos []*gitmanager.GitRepository

	mux          sync.RWMutex
	pollInterval time.Duration
	ticker       *time.Ticker

	repository ConfigRepository
	extractor  Extractor
	parser     Parser
	analyzer   Analyzer
}

type ConfigleamRepo struct {
	Url   string
	Token string
}

type ConfigleamServiceConfig struct {
	Repos []ConfigleamRepo
	Envs  []string
}

func New(cfg ConfigleamServiceConfig, parser Parser, extractor Extractor, repository ConfigRepository, analyzer Analyzer) *configleamService {
	gitrepos := []*gitmanager.GitRepository{}

	for _, r := range cfg.Repos {
		repo, err := gitmanager.NewGitRepository(r.Url, "main", r.Token, cfg.Envs)
		if err != nil {
			log.Fatalf("Fatal generating %s local git-repository", r.Url)
		}
		gitrepos = append(gitrepos, repo)
	}

	return &configleamService{
		gitrepos:     gitrepos,
		pollInterval: 5 * time.Second,
		mux:          sync.RWMutex{},
		repository:   repository,
		extractor:    extractor,
		parser:       parser,
		analyzer:     analyzer,
	}
}

func (s *configleamService) Run(ctx context.Context) {
	err := s.cloneAllRemoteRepos(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	s.buildConfigFromLocalRepos(ctx)

	go s.watchRemoteReposForUpdates()
}

func (s *configleamService) cloneAllRemoteRepos(_ context.Context) error {
	for _, gitrepo := range s.gitrepos {
		err := gitrepo.CloneRemoteRepo()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *configleamService) buildConfigFromLocalRepos(ctx context.Context) error {
	for _, gitrepo := range s.gitrepos {
		err := s.buildConfigFromLocalRepo(ctx, gitrepo)
		if err != nil {
			log.Printf("Error while building the config from a local repo: %e\n", err)
			return err
		}
	}

	return nil
}

func (s *configleamService) watchRemoteReposForUpdates() {
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

func (s *configleamService) buildConfigFromLocalRepo(ctx context.Context, gitrepo *gitmanager.GitRepository) error {
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
		configList, err := s.extractor.ExtractConfigList(gitrepo.Dir)
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

		log.Println("Generated new repo config!", repoConfig)

		err = s.repository.StoreConfig(ctx, repoConfig)
		if err != nil {
			log.Println("Error storing config:", err)
			return err
		}
	}

	return nil
}

func (s *configleamService) cleanLocalRepos() {
	log.Println("Cleaning local repositories...")

	for _, gitrepo := range s.gitrepos {
		err := gitrepo.RemoveLocalRepo()
		if err != nil {
			log.Printf("Error on removing local repo %s from dir %s", gitrepo.URL, gitrepo.Dir)
		}
	}
}

func (s *configleamService) Shutdown() {
	s.ticker.Stop()
	s.cleanLocalRepos()
}

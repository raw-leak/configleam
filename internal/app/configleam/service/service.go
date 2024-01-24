package service

import (
	"log"
	"sync"
	"time"
)

type configleamService struct {
	repos        []*repo
	mux          sync.RWMutex
	pollInterval time.Duration
	ticker       *time.Ticker
}

type ConfigleamRepo struct {
	Url   string
	Token string
}

type ConfigleamServiceConfig struct {
	Branches []string
	Repos    []ConfigleamRepo
}

func NewConfigleamService(cfg ConfigleamServiceConfig) *configleamService {
	repos := []*repo{}

	for _, branch := range cfg.Branches {
		for _, r := range cfg.Repos {
			repo, err := newRepo(r.Url, branch, r.Token)
			if err != nil {
				log.Fatalf("Fatal generating %s local repository", r.Url)
			}
			repos = append(repos, repo)
		}
	}

	return &configleamService{
		repos:        repos,
		pollInterval: 1 * time.Second,
		mux:          sync.RWMutex{},
	}
}

func (s *configleamService) Run() {
	err := s.cloneRemoteRepos()
	if err != nil {
		log.Fatalf(err.Error())
	}

	go s.startMonitoringRemoteRepos()
}

func (s *configleamService) cloneRemoteRepos() error {
	for _, repo := range s.repos {
		err := repo.CloneRemoteRepo()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *configleamService) applyUpdates(cfg bool) error {
	// TODO: save changes of specific repo only
	return nil
}

func (s *configleamService) startMonitoringRemoteRepos() {
	s.ticker = time.NewTicker(s.pollInterval)

	for range s.ticker.C {
		for _, repo := range s.repos {

			newHash, changed, err := repo.HasHashChanged()
			if err != nil {
				log.Println("Error checking for updates:", err)
				continue
			}

			if changed {
				log.Println("New changes detected, applying updates...")

				config, err := repo.GetLatestConfig()
				if err != nil {
					log.Println("Error reading latest changes:", err)
					continue
				}

				err = s.applyUpdates(config)
				if err != nil {
					log.Println("Error applying changes:", err)
					continue
				}

				repo.SetLastHash(newHash)
			}
		}
	}
}

func (s *configleamService) cleanLocalRepos() {
	log.Println("Cleaning local repositories...")

	for _, repo := range s.repos {
		err := repo.RemoveLocalRepo()
		if err != nil {
			log.Printf("Error on removing local repo %s from dir %s", repo.url, repo.dir)
		}
	}
}

func (s *configleamService) Shutdown() {
	s.ticker.Stop()
	s.cleanLocalRepos()
}

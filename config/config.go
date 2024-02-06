package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Bool bool

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// This allows Bool to be filled from a string in the environment variables.
func (b *Bool) UnmarshalText(text []byte) error {
	str := strings.ToLower(string(text))
	*b = str == "true" || str == "1" || str == "yes" || str == "on"
	return nil
}

type Config struct {
	Port     string `envconfig:"PORT"`
	Hostname string `envconfig:"HOSTNAME"`
	RepoType string `envconfig:"REPO_TYPE"`

	RedisAddrs    string `envconfig:"REDIS_ADDRS"`
	RedisUsername string `envconfig:"REDIS_USERNAME"`
	RedisPassword string `envconfig:"REDIS_PASSWORD"`

	EtcdAddrs    string `envconfig:"ETCD_ADDRS"`
	EtcdUsername string `envconfig:"ETCD_USERNAME"`
	EtcdPassword string `envconfig:"ETCD_PASSWORD"`

	LeaseLockName      string        `envconfig:"K8S_LEASE_LOCK_NAME" default:"configleam-lock"`
	LeaseLockNamespace string        `envconfig:"K8S_LEASE_LOCK_NAMESPACE" default:"default"`
	LeaseDuration      time.Duration `envconfig:"K8S_LEASE_DURATION"`
	RenewDeadline      time.Duration `envconfig:"K8S_RENEW_DEADLINE"`
	RetryPeriod        time.Duration `envconfig:"K8S_RETRY_PERIOD"`

	EnableLeaderElection Bool `envconfig:"K8S_ENABLE_LEADER_ELECTION"`

	RepoConfig RepoConfig
}

type RepoConfig struct {
	Repositories []string `json:"repositories"`
	Environments []string `json:"environments"`
	Branch       string   `json:"branch" default:"main"`
}

var (
	config Config
	once   sync.Once
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
}

// Get reads config from environment. Once.
func Get() (*Config, error) {
	once.Do(func() {
		err := envconfig.Process("", &config)
		if err != nil {
			log.Fatalf("Error processing config: %v", err)
		}

		if err := loadConfigFile("config.json", &config.RepoConfig); err != nil {
			log.Fatalf("Error loading config.json: %v", err)
		}
	})

	if len(config.RepoConfig.Repositories) < 1 {
		return nil, fmt.Errorf("there are no repositories provided")
	}

	if len(config.RepoConfig.Environments) < 1 {
		return nil, fmt.Errorf("there are no environments provided")
	}

	return &config, nil
}

// loadConfigFile reads and parses a JSON configuration file.
func loadConfigFile(filename string, cfg interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return err
	}

	return nil
}

package config

import (
	"log"
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
	Port         string        `envconfig:"PORT"`
	Hostname     string        `envconfig:"HOSTNAME"`
	Tls          Bool          `envconfig:"TLS" default:"true"`
	PullInterval time.Duration `envconfig:"CG_PULL_INTERVAL"`

	RedisAddrs    string `envconfig:"REDIS_ADDRS"`
	RedisUsername string `envconfig:"REDIS_USERNAME"`
	RedisPassword string `envconfig:"REDIS_PASSWORD"`
	RedisTls      Bool   `envconfig:"REDIS_TLS"`

	EtcdAddrs    []string `envconfig:"ETCD_ADDRS" delim:","`
	EtcdUsername string   `envconfig:"ETCD_USERNAME"`
	EtcdPassword string   `envconfig:"ETCD_PASSWORD"`

	// cfg repo
	RepoUrl    string   `envconfig:"GIT_REPOSITORY_URL"`
	RepoEnvs   []string `envconfig:"GIT_REPOSITORY_ENVS" delim:","`
	RepoBranch string   `envconfig:"GIT_REPOSITORY_BRANCH" default:"main"`

	// k8s
	LeaseLockName      string        `envconfig:"K8S_LEASE_LOCK_NAME" default:"configleam-lock"`
	LeaseLockNamespace string        `envconfig:"K8S_LEASE_LOCK_NAMESPACE" default:"default"`
	LeaseDuration      time.Duration `envconfig:"K8S_LEASE_DURATION"`
	RenewDeadline      time.Duration `envconfig:"K8S_RENEW_DEADLINE"`
	RetryPeriod        time.Duration `envconfig:"K8S_RETRY_PERIOD"`

	EnableLeaderElection Bool `envconfig:"K8S_ENABLE_LEADER_ELECTION"`
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
	})

	return &config, nil
}

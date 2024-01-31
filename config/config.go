package config

import (
	"log"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port  string `envconfig:"PORT"`
	Token string `envconfig:"GIT_TOKEN"`
	Url   string `envconfig:"GIT_URL"`

	RedisAddrs    string `envconfig:"REDIS_ADDRS"`
	RedisPassword string `envconfig:"REDIS_PASSWORD"`
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
func Get() *Config {
	once.Do(func() {
		err := envconfig.Process("", &config)
		if err != nil {
			log.Fatalf("Error processing config: %v", err)
		}

	})

	return &config
}

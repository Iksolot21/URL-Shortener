package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env            string `yaml:"env" env-default:"local"`
	StorageType    string `yaml:"storage_type" env-default:"memory"` // "memory" or "postgres"
	HTTPServer     `yaml:"http_server"`
	ShortURLLength int    `yaml:"short_url_length" env-default:"10"`
	PostgresURL    string `yaml:"postgres_url" env-default:"postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8082"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPath = "./config/local-memory.yaml"
		log.Printf("CONFIG_PATH is not set, using default path: %s", configPath)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file %s does not exist", configPath)
	} else if err != nil {
		log.Fatalf("error checking config file: %s", err)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	if cfg.StorageType == "" {
		log.Fatalf("StorageType is empty after reading config. Possible YAML parsing issue?")
	}

	return &cfg
}

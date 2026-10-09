package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	PORT int `env:"INGESTPOC_SERVER_PORT" envDefault:"8080"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	return cfg, nil
}

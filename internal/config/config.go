package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr        string
	DatabaseURL string
	FrontendURL string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        env("ADDR", "localhost:8080"),
		DatabaseURL: os.Getenv("POSTGRES_URL"),
		FrontendURL: env("FRONTEND_URL", "http://localhost:5173"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

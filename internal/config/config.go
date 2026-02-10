package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	FrontendURL   string
	StripeKey     string
	WebhookSecret string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:          env("ADDR", "localhost:8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		FrontendURL:   env("FRONTEND_URL", "http://localhost:5173"),
		StripeKey:     os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
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

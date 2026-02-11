package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv          string
	Addr            string
	DatabaseURL     string
	FrontendURL     string
	StorageDriver   string
	UploadsDir      string
	R2Endpoint      string
	R2Bucket        string
	R2AccessKeyID   string
	R2SecretKey     string
	R2PublicBaseURL string
	StripeKey       string
	WebhookSecret   string
}

func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_URL")
	}

	cfg := Config{
		AppEnv:          env("APP_ENV", "dev"),
		Addr:            env("ADDR", "localhost:8080"),
		DatabaseURL:     dbURL,
		FrontendURL:     env("FRONTEND_URL", "http://localhost:5173"),
		StorageDriver:   env("STORAGE_DRIVER", "local"),
		UploadsDir:      env("UPLOADS_DIR", "uploads"),
		R2Endpoint:      os.Getenv("R2_ENDPOINT"),
		R2Bucket:        os.Getenv("R2_BUCKET"),
		R2AccessKeyID:   os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretKey:     os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2PublicBaseURL: os.Getenv("R2_PUBLIC_BASE_URL"),
		StripeKey:       os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret:   os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL or POSTGRES_URL is required")
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

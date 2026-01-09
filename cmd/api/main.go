package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	httpserver "groupie-tracker/internal/http"
)

func main() {
	_ = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	artistRepo := artist.NewRepository(pool)
	artistHandler := artist.NewHandler(artistRepo)

	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler)
	log.Fatal(r.Run(cfg.Addr))
}

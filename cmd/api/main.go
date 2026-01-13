package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/album"
	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/concert"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	"groupie-tracker/internal/geo"
	httpserver "groupie-tracker/internal/http"
	"groupie-tracker/internal/track"
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

	albumRepo := album.NewRepository(pool)
	albumHandler := album.NewHandler(albumRepo)

	trackRepo := track.NewRepository(pool)
	trackHandler := track.NewHandler(trackRepo)

	concertRepo := concert.NewRepository(pool)
	concertHandler := concert.NewHandler(concertRepo)

	geoRepo := geo.NewRepository(pool)
	geoService := geo.NewService(geoRepo, geo.DummyGeocoder{})
	geoHandler := geo.NewHandler(geoRepo, geoService)

	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler, albumHandler, trackHandler, concertHandler, geoHandler)
	log.Fatal(r.Run(cfg.Addr))
}

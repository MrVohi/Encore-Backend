package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/album"
	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/concert"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	"groupie-tracker/internal/geo"
	httpserver "groupie-tracker/internal/http"
	"groupie-tracker/internal/media"
	"groupie-tracker/internal/search"
	"groupie-tracker/internal/track"
)

func main() {

	if err := godotenv.Load(".env"); err != nil {
		log.Println("Note: No .env file found or error loading it")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to artists database:", err)
	}
	defer pool.Close()

	var db, schema, addr string
	var port int
	err = pool.QueryRow(ctx, `SELECT current_database(), current_schema(), inet_server_addr()::text, inet_server_port()`).Scan(&db, &schema, &addr, &port)
	fmt.Println("CONNECTED TO:", db, schema, addr, port)

	artistRepo := artist.NewRepository(pool)
	mediaRepo := media.NewRepository(pool)
	mediaService := media.NewService(mediaRepo, "uploads")
	artistService := artist.NewService(artistRepo, mediaService)
	artistHandler := artist.NewHandler(artistRepo, artistService)

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
	api := r.Group("/api")
	searchRepo := search.NewRepository(pool)
	searchHandler := search.NewHandler(searchRepo)
	searchHandler.RegisterRoutes(api)

	log.Printf("Server starting on %s", cfg.Addr)
	log.Fatal(r.Run(cfg.Addr))
}

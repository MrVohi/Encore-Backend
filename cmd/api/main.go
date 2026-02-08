package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/album"
	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/concert"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	"groupie-tracker/internal/follow"
	"groupie-tracker/internal/geo"
	httpserver "groupie-tracker/internal/http"
	"groupie-tracker/internal/media"
	"groupie-tracker/internal/notifications"
	"groupie-tracker/internal/search"
	"groupie-tracker/internal/storage"
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

	var store storage.Driver
	switch strings.ToLower(cfg.StorageDriver) {
	case "r2":
		store, err = storage.NewR2Storage(ctx, cfg.R2Endpoint, cfg.R2Bucket, cfg.R2AccessKeyID, cfg.R2SecretKey)
		if err != nil {
			log.Fatal("Failed to configure R2 storage:", err)
		}
	case "local", "":
		store = storage.NewLocalStorage(cfg.UploadsDir)
	default:
		log.Fatalf("Unsupported STORAGE_DRIVER: %s", cfg.StorageDriver)
	}

	mediaService := media.NewService(mediaRepo, store)
	artistService := artist.NewService(artistRepo, mediaService)
	artistHandler := artist.NewHandler(artistRepo, artistService, cfg.R2PublicBaseURL)

	albumRepo := album.NewRepository(pool)
	albumHandler := album.NewHandler(albumRepo)

	trackRepo := track.NewRepository(pool)
	trackHandler := track.NewHandler(trackRepo)

	geoRepo := geo.NewRepository(pool)
	geoService := geo.NewService(geoRepo, geo.DummyGeocoder{})
	geoHandler := geo.NewHandler(geoRepo, geoService)

	followRepo := follow.NewRepository(pool)
	followHandler := follow.NewHandler(followRepo)

	notifyRepo := notifications.NewRepository(pool)
	notifySender := notifications.NewSMTPSenderFromEnv()
	notifyService := notifications.NewService(notifyRepo, notifySender, cfg.FrontendURL)
	notifyService.Start(ctx)

	concertRepo := concert.NewRepository(pool)
	concertHandler := concert.NewHandler(concertRepo, notifyService)

	r := httpserver.NewRouter(cfg.FrontendURL, cfg.UploadsDir, artistHandler, albumHandler, trackHandler, concertHandler, geoHandler, followHandler)
	api := r.Group("/api")
	searchRepo := search.NewRepository(pool)
	searchHandler := search.NewHandler(searchRepo)
	searchHandler.RegisterRoutes(api)

	log.Printf("Server starting on %s", cfg.Addr)
	log.Fatal(r.Run(cfg.Addr))
}

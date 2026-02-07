package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/concert"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	"groupie-tracker/internal/follow"
	httpserver "groupie-tracker/internal/http"
	"groupie-tracker/internal/notifications"
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
	artistHandler := artist.NewHandler(artistRepo)

	followRepo := follow.NewRepository(pool)
	followHandler := follow.NewHandler(followRepo)

	notifyRepo := notifications.NewRepository(pool)
	notifySender := notifications.NewSMTPSenderFromEnv()
	notifyService := notifications.NewService(notifyRepo, notifySender, cfg.FrontendURL)
	notifyService.Start(ctx)

	concertRepo := concert.NewRepository(pool)
	concertHandler := concert.NewHandler(concertRepo, notifyService)

	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler, followHandler, concertHandler)

	log.Printf("Server starting on %s", cfg.Addr)
	log.Fatal(r.Run(cfg.Addr))
}

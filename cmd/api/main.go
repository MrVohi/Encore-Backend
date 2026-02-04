package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	httpserver "groupie-tracker/internal/http"
)

func main() {

	if err := godotenv.Load(".env"); err != nil {
		log.Println("Note: No .env file found or error loading it")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Erreur chargement config:", err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Erreur connexion DB Artistes:", err)
	}
	defer pool.Close()

	var db, schema, addr string
	var port int
	err = pool.QueryRow(ctx, `SELECT current_database(), current_schema(), inet_server_addr()::text, inet_server_port()`).Scan(&db, &schema, &addr, &port)
	fmt.Println("CONNECTED TO:", db, schema, addr, port)

	artistRepo := artist.NewRepository(pool)
	artistHandler := artist.NewHandler(artistRepo)

	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler)

	log.Printf("Server starting on %s", cfg.Addr)
	log.Fatal(r.Run(cfg.Addr))
}

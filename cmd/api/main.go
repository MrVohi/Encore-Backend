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

	"groupie-tracker/internal/authentification/auth"
	"groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/media"
	"groupie-tracker/internal/middleware"
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

	database.Connect()

	artistRepo := artist.NewRepository(pool)
	mediaRepo := media.NewRepository(pool)
	mediaService := media.NewService(mediaRepo, "uploads")
	artistService := artist.NewService(artistRepo, mediaService)
	artistHandler := artist.NewHandler(artistRepo, artistService)

	authHandler := auth.NewAuthHandler()

	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler)
	r.Static("/uploads", "./uploads")

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		{

			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.GET("/verify-email", authHandler.VerifyEmail)
			authGroup.POST("/forgot-password", authHandler.RequestPasswordReset)
			authGroup.POST("/reset-password", authHandler.ResetPassword)
			authGroup.GET("/google", authHandler.GoogleLogin)
			authGroup.GET("/google/callback", authHandler.GoogleCallback)
			authGroup.POST("/refresh", authHandler.RefreshToken)
		}

		api.GET("/users", authHandler.ListUsers)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", authHandler.GetCurrentUser)
		}
	}

	log.Printf("Server starting on %s", cfg.Addr)
	log.Fatal(r.Run(cfg.Addr))
}

// package main

// import (
// 	"context"
// 	"log"

// 	"github.com/joho/godotenv"

// 	"groupie-tracker/internal/artist"
// 	"groupie-tracker/internal/config"
// 	"groupie-tracker/internal/db"
// 	httpserver "groupie-tracker/internal/http"
// )

// func main() {
// 	_ = godotenv.Load(".env")

// 	cfg, err := config.Load()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	ctx := context.Background()

// 	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer pool.Close()

// 	artistRepo := artist.NewRepository(pool)
// 	artistHandler := artist.NewHandler(artistRepo)

// 	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler)
// 	log.Fatal(r.Run(cfg.Addr))

// }

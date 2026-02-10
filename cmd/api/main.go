package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/concerts"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	httpserver "groupie-tracker/internal/http"
	"groupie-tracker/internal/tickets"

	"groupie-tracker/internal/authentification/auth"
	"groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/middleware"

	"github.com/stripe/stripe-go/v84"
)

func main() {

	if err := godotenv.Load(".env"); err != nil {
		log.Println("Note: No .env file found or error loading it")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Erreur chargement config:", err)
	}

	if cfg.StripeKey != "" {
		stripe.Key = cfg.StripeKey
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
	artistHandler := artist.NewHandler(artistRepo)
	concertRepo := concerts.NewRepository(pool)
	concertHandler := concerts.NewHandler(concertRepo)
	ticketsRepo := tickets.NewRepository(pool)
	ticketsHandler := tickets.NewHandler(ticketsRepo, cfg.FrontendURL, cfg.WebhookSecret)

	authHandler := auth.NewAuthHandler()

	r := httpserver.NewRouter(cfg.FrontendURL, artistHandler)

	api := r.Group("/api")
	{
		ticketsHandler.RegisterPublicRoutes(api)
		ticketsHandler.RegisterWebhookRoutes(api)
		concertHandler.RegisterPublicRoutes(api)

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

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", authHandler.GetCurrentUser)
			ticketsHandler.RegisterProtectedRoutes(protected)
		}

		admin := api.Group("/")
		admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
		{
			concertHandler.RegisterAdminRoutes(admin)
			ticketsHandler.RegisterAdminRoutes(admin)
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

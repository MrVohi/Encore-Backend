package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v84"

	"groupie-tracker/internal/album"
	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/cart"
	"groupie-tracker/internal/concert"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/db"
	"groupie-tracker/internal/follow"
	"groupie-tracker/internal/geo"
	httpserver "groupie-tracker/internal/http"
	"groupie-tracker/internal/media"
	"groupie-tracker/internal/middleware"
	"groupie-tracker/internal/notifications"
	"groupie-tracker/internal/search"
	"groupie-tracker/internal/storage"
	"groupie-tracker/internal/tickets"
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

	dsn := strings.TrimSpace(os.Getenv("SENTRY_DSN"))
	if dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      os.Getenv("SENTRY_ENVIRONMENT"),
			Release:          os.Getenv("SENTRY_RELEASE"),
			AttachStacktrace: true,
			TracesSampleRate: 0.0,
		}); err != nil {
			log.Printf("Sentry init failed: %v\n", err)
		} else {
			defer sentry.Flush(2 * time.Second)
		}
	}

	if cfg.StripeKey != "" {
		stripe.Key = cfg.StripeKey
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to artists database:", err)
	}
	defer pool.Close()

	var dbName, schema, addr string
	var port int
	err = pool.QueryRow(ctx, `SELECT current_database(), current_schema(), inet_server_addr()::text, inet_server_port()`).Scan(&dbName, &schema, &addr, &port)
	fmt.Println("CONNECTED TO:", dbName, schema, addr, port)

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

	ticketsRepo := tickets.NewRepository(pool)
	ticketsHandler := tickets.NewHandler(ticketsRepo, cfg.FrontendURL, cfg.WebhookSecret, notifySender)
	cartRepo := cart.NewRepository(pool)
	cartHandler := cart.NewHandler(cartRepo, cfg.FrontendURL)

	r := httpserver.NewRouter(cfg.FrontendURL, cfg.UploadsDir, artistHandler, albumHandler, trackHandler, concertHandler, geoHandler, followHandler)
	api := r.Group("/api")

	searchRepo := search.NewRepository(pool)
	searchHandler := search.NewHandler(searchRepo)
	searchHandler.RegisterRoutes(api)

	ticketsHandler.RegisterPublicRoutes(api)
	ticketsHandler.RegisterWebhookRoutes(api)
	cartHandler.RegisterRoutes(api)

	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		ticketsHandler.RegisterProtectedRoutes(protected)
	}

	admin := api.Group("/")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		ticketsHandler.RegisterAdminRoutes(admin)
	}

	r.GET("/api/sentry-test", func(c *gin.Context) {
		sentry.CaptureMessage("Backend Sentry Test (safe to ignore)")
		c.JSON(200, gin.H{"ok": true})
	})

	log.Printf("Server starting on %s", cfg.Addr)
	log.Fatal(r.Run(cfg.Addr))
}

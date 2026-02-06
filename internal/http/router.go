package httpserver

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/album"
	"groupie-tracker/internal/artist"
	auth "groupie-tracker/internal/authentification/auth"
	authdb "groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/concert"
	"groupie-tracker/internal/geo"
	"groupie-tracker/internal/http/middleware"
	"groupie-tracker/internal/track"
)

func NewRouter(frontendURL string, artistHandler *artist.Handler, albumHandler *album.Handler, trackHandler *track.Handler, concertHandler *concert.Handler, geoHandler *geo.Handler) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(middleware.CORS(frontendURL)))
	r.Static("/uploads", "./uploads")

	api := r.Group("/api")
	// register artist routes
	artistHandler.RegisterRoutes(api)
	albumHandler.RegisterRoutes(api)
	trackHandler.RegisterRoutes(api)
	concertHandler.RegisterRoutes(api)
	geoHandler.RegisterRoutes(api)

	// register health route
	api.GET("/health", func(c *gin.Context) {
		// lightweight import to avoid cycle; call handler directly
		groupieHealth := func(c *gin.Context) { c.JSON(200, gin.H{"server": "ok"}) }
		groupieHealth(c)
	})

	// setup auth DB and register auth routes
	authdb.Connect()
	auth.RegisterRoutes(api)

	return r
}

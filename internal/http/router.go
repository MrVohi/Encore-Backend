package httpserver

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/artist"
	auth "groupie-tracker/internal/authentification/auth"
	authdb "groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/http/middleware"
)

func NewRouter(frontendURL string, artistHandler *artist.Handler) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(middleware.CORS(frontendURL)))

	api := r.Group("/api")
	// register artist routes
	artistHandler.RegisterRoutes(api)

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

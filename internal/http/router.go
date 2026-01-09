package httpserver

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/http/middleware"
)

func NewRouter(frontendURL string, artistHandler *artist.Handler) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(middleware.CORS(frontendURL)))

	api := r.Group("/api")
	artistHandler.RegisterRoutes(api)

	return r
}

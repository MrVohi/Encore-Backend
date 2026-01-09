package httpserver

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/track"
	"groupie-tracker/internal/album"
	"groupie-tracker/internal/artist"
	"groupie-tracker/internal/http/middleware"
)

func NewRouter(frontendURL string, artistHandler *artist.Handler, albumHandler *album.Handler, trackHandler *track.Handler) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(middleware.CORS(frontendURL)))

	api := r.Group("/api")
	artistHandler.RegisterRoutes(api)
	albumHandler.RegisterRoutes(api)
	trackHandler.RegisterRoutes(api)

	return r
}

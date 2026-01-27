package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
)

func CORS(frontendURL string) cors.Config {
	return cors.Config{
		AllowOrigins: []string{
			frontendURL,
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

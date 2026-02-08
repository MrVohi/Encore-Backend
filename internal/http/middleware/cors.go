package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
)

func CORS(frontendURL string) cors.Config {
	return cors.Config{
		AllowOrigins: []string{
			frontendURL,
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://192.168.1.17:5173",
		},
		AllowOriginFunc: func(origin string) bool {
			if origin == frontendURL {
				return true
			}
			if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" || origin == "http://192.168.1.17:5173" {
				return true
			}
			// Allow common LAN dev origins on port 5173
			if strings.HasPrefix(origin, "http://192.168.") && strings.HasSuffix(origin, ":5173") {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

package middleware

import "github.com/gin-contrib/cors"

func CORS(frontendURL string) cors.Config {
	return cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return origin == frontendURL || origin == "http://127.0.0.1:5173"
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}
}

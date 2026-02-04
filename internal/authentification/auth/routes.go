package auth

import (
	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/middleware"
)

// RegisterRoutes registers auth endpoints under /api/auth and protected routes.
func RegisterRoutes(api *gin.RouterGroup) {
	authHandler := NewAuthHandler()

	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// Keep existing /api/auth/me for backwards compatibility
	protected := api.Group("/auth")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", authHandler.GetCurrentUser)
		protected.POST("/logout", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "logged out"})
		})
	}

	// Also expose top-level protected routes for frontend compatibility (/api/me)
	protectedRoot := api.Group("/")
	protectedRoot.Use(middleware.AuthMiddleware())
	{
		protectedRoot.GET("/me", authHandler.GetCurrentUser)
		protectedRoot.POST("/logout", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "logged out"})
		})
	}
}

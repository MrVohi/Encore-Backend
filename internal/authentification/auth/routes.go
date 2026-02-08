package auth

import (
	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/middleware"
)

// RegisterRoutes registers auth endpoints under /api/auth and protected routes.
func RegisterRoutes(api *gin.RouterGroup) {
	authHandler := NewAuthHandler()
	BackfillGoogleUsers()

	adminUsers := api.Group("/users")
	adminUsers.Use(middleware.AuthMiddleware())
	adminUsers.Use(middleware.AdminOnly())
	{
		adminUsers.GET("", authHandler.ListUsers)
		adminUsers.POST("/:id/promote", authHandler.PromoteUser)
		adminUsers.DELETE("/:id", authHandler.DeleteUser)
	}

	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.GET("/google", authHandler.GoogleLogin)
		authGroup.GET("/google/callback", authHandler.GoogleCallback)
		authGroup.POST("/forgot-password", authHandler.RequestPasswordReset)
		authGroup.POST("/reset-password", authHandler.ResetPassword)
		authGroup.GET("/verify-email", authHandler.VerifyEmail)
		authGroup.POST("/resend-verification", authHandler.ResendVerification)
	}

	// Keep existing /api/auth/me for backwards compatibility
	protected := api.Group("/auth")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", authHandler.GetCurrentUser)
		protected.PUT("/profile", authHandler.UpdateProfile)
		protected.POST("/password", authHandler.ChangePassword)
		protected.POST("/avatar", authHandler.UploadAvatar)
		protected.DELETE("/avatar", authHandler.DeleteAvatar)
		protected.POST("/logout", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "logged out"})
		})
	}

	// Also expose top-level protected routes for frontend compatibility (/api/me)
	protectedRoot := api.Group("/")
	protectedRoot.Use(middleware.AuthMiddleware())
	{
		protectedRoot.GET("/me", authHandler.GetCurrentUser)
		protectedRoot.GET("/user", authHandler.GetCurrentUser)
		protectedRoot.POST("/logout", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "logged out"})
		})
	}
}

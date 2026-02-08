package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/authentification/models"
)

// AdminOnly ensures the authenticated user has admin role.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("user_id")
		if !ok || userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			c.Abort()
			return
		}

		db := database.GetDB()
		if db == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Select("id, role").First(&user, "id = ?", userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
			c.Abort()
			return
		}
		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}

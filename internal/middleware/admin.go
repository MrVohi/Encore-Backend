package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminMiddleware() gin.HandlerFunc {
	allowedEmails := toSet(os.Getenv("ADMIN_EMAILS"))
	allowedIDs := toSet(os.Getenv("ADMIN_USER_IDS"))

	allowAll := allowedEmails["*"] || allowedIDs["*"]

	return func(c *gin.Context) {
		if allowAll || (len(allowedEmails) == 0 && len(allowedIDs) == 0) {
			c.Next()
			return
		}

		var email string
		if v, ok := c.Get("user_email"); ok {
			if s, ok := v.(string); ok {
				email = strings.ToLower(strings.TrimSpace(s))
			}
		}

		var userID string
		if v, ok := c.Get("user_id"); ok {
			if s, ok := v.(string); ok {
				userID = strings.TrimSpace(s)
			}
		}

		if (email != "" && allowedEmails[email]) || (userID != "" && allowedIDs[userID]) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		c.Abort()
	}
}

func toSet(raw string) map[string]bool {
	set := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed == "" {
			continue
		}
		set[trimmed] = true
	}
	return set
}

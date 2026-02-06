package health

import (
	"net/http"
	"time"

	"groupie-tracker/internal/authentification/database"

	"github.com/gin-gonic/gin"
)

// HealthHandler returns basic server + DB status
func HealthHandler(c *gin.Context) {
	db := database.GetDB()
	status := gin.H{"server": "ok", "db": "unknown"}

	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			// quick ping with timeout
			pingCh := make(chan error, 1)
			go func() { pingCh <- sqlDB.Ping() }()
			select {
			case err := <-pingCh:
				if err == nil {
					status["db"] = "ok"
				} else {
					status["db"] = "error"
				}
			case <-time.After(500 * time.Millisecond):
				status["db"] = "timeout"
			}
		}
	}

	c.JSON(http.StatusOK, status)
}

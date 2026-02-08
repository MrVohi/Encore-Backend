package concert

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"groupie-tracker/internal/middleware"
	"groupie-tracker/internal/notifications"
)

type Notifier interface {
	EnqueueConcert(job notifications.ConcertJob)
}

type Handler struct {
	repo     *Repository
	notifier Notifier
}

func NewHandler(repo *Repository, notifier Notifier) *Handler {
	return &Handler{repo: repo, notifier: notifier}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/concerts", h.list)
	rg.GET("/concerts/:id", h.getByID)
	rg.GET("/artists/:id/concerts", h.listArtistConcerts)

	admin := rg.Group("/")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		admin.POST("/artists/:id/concerts", h.create)
		admin.DELETE("/concerts/:id", h.delete)
	}
}

func (h *Handler) list(c *gin.Context) {
	concerts, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, concerts)
}

func (h *Handler) listArtistConcerts(c *gin.Context) {
	concerts, err := h.repo.ListByArtist(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, concerts)
}

func (h *Handler) getByID(c *gin.Context) {
	a, err := h.repo.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "concert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) create(c *gin.Context) {
	artistID := c.Param("id")
	exists, err := h.repo.ArtistExists(c.Request.Context(), artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		return
	}

	var req CreateConcertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.repo.Create(c.Request.Context(), req, artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.notifier != nil {
		artistName, err := h.repo.ArtistName(c.Request.Context(), artistID)
		if err == nil {
			when, parseErr := time.Parse("2006-01-02 15:04:05", created.When)
			if parseErr == nil {
				h.notifier.EnqueueConcert(notifications.ConcertJob{
					ConcertID:  created.ID,
					ArtistID:   artistID,
					ArtistName: artistName,
					When:       when,
					City:       created.City,
					Country:    created.Country,
				})
			}
		} else if err != nil && err != pgx.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.repo.DeleteByID(c.Request.Context(), c.Param("id")); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "concert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

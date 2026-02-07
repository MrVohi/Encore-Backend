package concert

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

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
	rg.GET("/artists/:id/concerts", h.listByArtist)
	rg.POST("/artists/:id/concerts", h.create)
}

func (h *Handler) listByArtist(c *gin.Context) {
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

	concerts, err := h.repo.ListByArtist(c.Request.Context(), artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, concerts)
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

	created, err := h.repo.Create(c.Request.Context(), artistID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.notifier != nil {
		artistName, err := h.repo.ArtistName(c.Request.Context(), artistID)
		if err == nil {
			h.notifier.EnqueueConcert(notifications.ConcertJob{
				ConcertID:  created.ID,
				ArtistID:   artistID,
				ArtistName: artistName,
				When:       created.When,
				City:       created.City,
				Country:    created.Country,
			})
		} else if err != nil && err != pgx.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, created)
}

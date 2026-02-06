package track

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/tracks/:id", h.getByID)
	rg.GET("/albums/:id/tracks", h.listAlbumTracks)
	rg.POST("/albums/:id/tracks", h.createAlbumTracks)
}

func (h *Handler) listAlbumTracks(c *gin.Context) {
	tracks, err := h.repo.ListAlbumTracks(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tracks)
}

func (h *Handler) getByID(c *gin.Context) {
	a, err := h.repo.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) createAlbumTracks(c *gin.Context) {
	var req CreateTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.repo.createAlbumTracks(c.Request.Context(), req, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, a)
}

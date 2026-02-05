package artist

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"groupie-tracker/internal/media"
)

type Handler struct {
	repo    *Repository
	service *Service
}

func NewHandler(repo *Repository, service *Service) *Handler {
	return &Handler{
		repo:    repo,
		service: service,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/artists", h.list)
	rg.GET("/artists/:id", h.getByID)
	rg.POST("/artists", h.create)
	rg.POST("/artists/:id/artwork", h.uploadArtwork)
	rg.POST("/artists/:id/preview", h.uploadPreview)
	rg.DELETE("/artists/:id", h.delete)
}

func (h *Handler) list(c *gin.Context) {
	artists, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for i := range artists {
		h.resolveURLs(c, &artists[i])
	}
	c.JSON(http.StatusOK, artists)
}

func (h *Handler) getByID(c *gin.Context) {
	a, err := h.repo.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.resolveURLs(c, &a)
	c.JSON(http.StatusOK, a)
}

func (h *Handler) create(c *gin.Context) {
	var req CreateArtistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.resolveURLs(c, &a)
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) uploadArtwork(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	a, err := h.service.UploadArtwork(c.Request.Context(), c.Param("id"), fileHeader)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}
		if errors.Is(err, media.ErrInvalidMimeType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.resolveURLs(c, &a)
	c.JSON(http.StatusOK, a)
}

func (h *Handler) uploadPreview(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	a, err := h.service.UploadPreview(c.Request.Context(), c.Param("id"), fileHeader)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}
		if errors.Is(err, media.ErrInvalidMimeType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.resolveURLs(c, &a)
	c.JSON(http.StatusOK, a)
}

func (h *Handler) delete(c *gin.Context) {
	err := h.service.DeleteArtist(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) resolveURLs(c *gin.Context, a *Artist) {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := c.Request.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	host := c.Request.Host
	if host == "" {
		host = "localhost:8080"
	}
	a.ResolveURLs(scheme + "://" + host)
}

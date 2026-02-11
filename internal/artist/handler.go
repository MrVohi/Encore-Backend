package artist

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"groupie-tracker/internal/media"
	"groupie-tracker/internal/middleware"
	"groupie-tracker/pkg/utils"
)

type Handler struct {
	repo         *Repository
	service      *Service
	r2PublicBase string
}

func NewHandler(repo *Repository, service *Service, r2PublicBase string) *Handler {
	return &Handler{
		repo:         repo,
		service:      service,
		r2PublicBase: r2PublicBase,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/artists", h.list)
	rg.GET("/artists/:id", h.getByID)

	admin := rg.Group("/")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		admin.POST("/artists", h.create)
		admin.PUT("/artists/:id", h.update)
		admin.POST("/artists/:id/artwork", h.uploadArtwork)
		admin.POST("/artists/:id/preview", h.uploadPreview)
		admin.DELETE("/artists/:id", h.delete)
	}
}

func (h *Handler) list(c *gin.Context) {
	userID := optionalUserID(c)
	genre := strings.TrimSpace(c.Query("genre"))
	var genrePtr *string
	if genre != "" {
		genrePtr = &genre
	}
	artists, err := h.repo.List(c.Request.Context(), userID, genrePtr)
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
	userID := optionalUserID(c)
	a, err := h.repo.GetByID(c.Request.Context(), c.Param("id"), userID)
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

	// Ensure non-null values for legacy columns when files are uploaded separately
	if req.ImageURL == nil {
		empty := ""
		req.ImageURL = &empty
	}
	if req.PreviewURL == nil {
		empty := ""
		req.PreviewURL = &empty
	}

	a, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.resolveURLs(c, &a)
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) update(c *gin.Context) {
	var req UpdateArtistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == nil && req.Genre == nil && req.ImageURL == nil && req.PreviewURL == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	a, err := h.repo.Update(c.Request.Context(), c.Param("id"), req)
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
	a.ResolveURLs(scheme+"://"+host, h.r2PublicBase)
}

func optionalUserID(c *gin.Context) *string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil
	}

	claims, err := utils.ValidateToken(parts[1])
	if err != nil || claims.UserID == "" {
		return nil
	}

	id := claims.UserID
	return &id
}

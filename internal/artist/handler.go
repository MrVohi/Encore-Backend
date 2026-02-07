package artist

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"groupie-tracker/pkg/utils"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/artists", h.list)
	rg.GET("/artists/:id", h.getByID)
	rg.POST("/artists", h.create)
}

func (h *Handler) list(c *gin.Context) {
	userID := optionalUserID(c)
	artists, err := h.repo.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	c.JSON(http.StatusCreated, a)
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

package concerts

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/concerts", h.list)
	rg.GET("/concerts/:id", h.getByID)
	rg.GET("/artists/:id/concerts", h.listByArtist)
}

func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/artists/:id/concerts", h.create)
	rg.PUT("/concerts/:id", h.update)
	rg.DELETE("/concerts/:id", h.delete)
}

func (h *Handler) list(c *gin.Context) {
	concerts, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load concerts"})
		return
	}
	c.JSON(http.StatusOK, concerts)
}

func (h *Handler) listByArtist(c *gin.Context) {
	artistID := strings.TrimSpace(c.Param("id"))
	if artistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artist id is required"})
		return
	}
	concerts, err := h.repo.ListByArtist(c.Request.Context(), artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load concerts"})
		return
	}
	c.JSON(http.StatusOK, concerts)
}

func (h *Handler) getByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "concert id is required"})
		return
	}
	concert, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "concert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load concert"})
		return
	}
	c.JSON(http.StatusOK, concert)
}

func (h *Handler) create(c *gin.Context) {
	artistID := strings.TrimSpace(c.Param("id"))
	if artistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artist id is required"})
		return
	}

	var req CreateConcertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if strings.TrimSpace(req.When) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "when is required"})
		return
	}
	if strings.TrimSpace(req.City) == "" || strings.TrimSpace(req.Country) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city and country are required"})
		return
	}

	concert, err := h.repo.Create(c.Request.Context(), artistID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create concert"})
		return
	}
	c.JSON(http.StatusCreated, concert)
}

func (h *Handler) update(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "concert id is required"})
		return
	}

	var req UpdateConcertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if strings.TrimSpace(req.When) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "when is required"})
		return
	}
	if strings.TrimSpace(req.City) == "" || strings.TrimSpace(req.Country) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city and country are required"})
		return
	}

	concert, err := h.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "concert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update concert"})
		return
	}
	c.JSON(http.StatusOK, concert)
}

func (h *Handler) delete(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "concert id is required"})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "concert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete concert"})
		return
	}
	c.Status(http.StatusNoContent)
}

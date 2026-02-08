package geo

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/middleware"
)

type Handler struct {
	repo    *Repository
	service *Service
}

func NewHandler(repo *Repository, service *Service) *Handler {
	return &Handler{repo: repo, service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/geo", h.get)

	admin := rg.Group("/")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		admin.POST("/geo/resolve", h.resolve)
	}
}

func (h *Handler) get(c *gin.Context) {
	city := c.Query("city")
	country := c.Query("country")

	cityN := normalize(city)
	countryN := normalize(country)

	lat, lng, ok, err := h.repo.Get(c.Request.Context(), cityN, countryN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"city": cityN, "country": countryN, "lat": lat, "lng": lng,
	})
}

type resolveRequest struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

func (h *Handler) resolve(c *gin.Context) {
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lat, lng, err := h.service.ResolveCoords(c.Request.Context(), req.City, req.Country)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"city":    normalize(req.City),
		"country": normalize(req.Country),
		"lat":     lat,
		"lng":     lng,
	})
}

package follow

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"groupie-tracker/internal/middleware"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	protected := rg.Group("/follows")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("", h.follow)
		protected.DELETE("/:artist_id", h.unfollow)
		protected.GET("", h.listFollowed)
		protected.GET("/:artist_id", h.isFollowing)
	}

	public := rg.Group("/follows")
	{
		public.GET("/:artist_id/followers", h.listFollowers)
	}
}

type followRequest struct {
	ArtistID string `json:"artist_id" binding:"required"`
}

func (h *Handler) follow(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req followRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exists, err := h.repo.ArtistExists(c.Request.Context(), req.ArtistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		return
	}

	if err := h.repo.Follow(c.Request.Context(), userID, req.ArtistID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}

func (h *Handler) unfollow(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	artistID := c.Param("artist_id")
	exists, err := h.repo.ArtistExists(c.Request.Context(), artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		return
	}

	if err := h.repo.Unfollow(c.Request.Context(), userID, artistID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

func (h *Handler) listFollowed(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ids, err := h.repo.ListFollowed(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"artist_ids": ids})
}

func (h *Handler) isFollowing(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	artistID := c.Param("artist_id")
	following, err := h.repo.IsFollowing(c.Request.Context(), userID, artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"artist_id": artistID, "following": following})
}

func (h *Handler) listFollowers(c *gin.Context) {
	artistID := c.Param("artist_id")
	exists, err := h.repo.ArtistExists(c.Request.Context(), artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		return
	}

	ids, err := h.repo.ListFollowers(c.Request.Context(), artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artist_id": artistID,
		"count":     len(ids),
		"followers": ids,
	})
}

func userIDFromContext(c *gin.Context) (string, bool) {
	userIDValue, _ := c.Get("user_id")
	userID, ok := userIDValue.(string)
	return userID, ok && userID != ""
}

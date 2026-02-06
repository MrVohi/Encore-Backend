package auth

import (
	"bytes"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"strings"

	"groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/authentification/models"
	"groupie-tracker/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	service *AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		service: NewAuthService(),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Register(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Login(input)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing token"})
		return
	}

	err := h.service.VerifyEmail(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		Mail  string `json:"mail"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		email = strings.TrimSpace(input.Mail)
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	_ = h.service.ResendVerification(email)

	c.JSON(http.StatusOK, gin.H{"message": "If this email exists, a verification link has been sent"})
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		Mail  string `json:"mail"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		email = strings.TrimSpace(input.Mail)
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	err := h.service.RequestPasswordReset(email)
	if err != nil {

		c.JSON(http.StatusOK, gin.H{"message": "If this email exists, a reset link has been sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If this email exists, a reset link has been sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var input struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.ResetPassword(input.Token, input.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = "random-state"
	}

	url := h.service.GetGoogleLoginURL(state)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code"})
		return
	}

	result, err := h.service.GoogleCallback(code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.RefreshAccessToken(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var input struct {
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if strings.TrimSpace(input.Username) != "" {
		candidate := sanitizeUsername(input.Username)
		if candidate == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username"})
			return
		}
		if !strings.EqualFold(candidate, user.Username) {
			ok, err := usernameAvailable(db, candidate)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate username"})
				return
			}
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "username already taken"})
				return
			}
			user.Username = candidate
		} else if candidate != user.Username {
			user.Username = candidate
		}
	}

	if strings.TrimSpace(input.FirstName) != "" {
		user.FirstName = strings.TrimSpace(input.FirstName)
	}
	if strings.TrimSpace(input.LastName) != "" {
		user.LastName = strings.TrimSpace(input.LastName)
	}
	if user.FirstName != "" || user.LastName != "" {
		user.Name = strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var input struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.Provider == "google" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password changes are not available for Google accounts"})
		return
	}

	if !utils.CheckPassword(input.CurrentPassword, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	user.PasswordHash = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	userID, _ := c.Get("user_id")

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read avatar"})
		return
	}
	defer src.Close()

	header := make([]byte, 512)
	n, _ := src.Read(header)
	contentType := http.DetectContentType(header[:n])

	_, err = src.Seek(0, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read avatar"})
		return
	}

	ext := ""
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image type"})
		return
	}

	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image too large (max 5MB)"})
		return
	}

	uploadsDir := filepath.Join(".", "uploads", "avatars")
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
		return
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	dstPath := filepath.Join(uploadsDir, filename)

	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
		return
	}
	if err := os.WriteFile(dstPath, buf.Bytes(), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
		return
	}

	avatarURL := "/uploads/avatars/" + filename

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.AvatarURL = avatarURL
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) DeleteAvatar(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.AvatarURL != "" {
		filename := path.Base(user.AvatarURL)
		if filename != "." && filename != "/" && filename != "" {
			avatarPath := filepath.Join(".", "uploads", "avatars", filename)
			_ = os.Remove(avatarPath)
		}
	}

	user.AvatarURL = ""
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete avatar"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

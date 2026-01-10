package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/authentification/email"
	"groupie-tracker/internal/authentification/models"
	"groupie-tracker/pkg/utils"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type AuthService struct {
	emailService *email.EmailService
	googleConfig *oauth2.Config
}

func NewAuthService() *AuthService {
	googleConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{
		emailService: email.NewEmailService(),
		googleConfig: googleConfig,
	}
}

type RegisterInput struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	FirstName      string `json:"first_name" binding:"required"`
	LastName       string `json:"last_name" binding:"required"`
	RecaptchaToken string `json:"recaptcha_token" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

func (s *AuthService) VerifyRecaptcha(token string) (bool, error) {
	secret := os.Getenv("RECAPTCHA_SECRET")
	url := fmt.Sprintf("https://www.google.com/recaptcha/api/siteverify?secret=%s&response=%s", secret, token)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result struct {
		Success bool    `json:"success"`
		Score   float64 `json:"score"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, err
	}

	// Pour reCAPTCHA v3, vérifier le score (0.0 à 1.0)
	// Score > 0.5 est généralement considéré comme humain

	return result.Success && result.Score > 0.5, nil
}

func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
	db := database.GetDB()

	isHuman, err := s.VerifyRecaptcha(input.RecaptchaToken)
	if err != nil || !isHuman {
		return nil, errors.New("échec de la vérification reCAPTCHA")
	}

	var existingUser models.User
	result := db.Where("email = ?", input.Email).First(&existingUser)
	if result.Error == nil {
		return nil, errors.New("cet e-mail est déjà utilisé")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	verifyToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Email:            input.Email,
		Password:         hashedPassword,
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		EmailVerifyToken: verifyToken,
		IsEmailVerified:  false,
		Provider:         "local",
	}

	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	go s.emailService.SendVerificationEmail(user.Email, verifyToken)

	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshTokenRecord := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(168 * time.Hour),
	}
	db.Create(&refreshTokenRecord)

	user.Password = ""

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func (s *AuthService) Login(input LoginInput) (*AuthResponse, error) {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("e-mail ou mot de passe incorrect")
		}
		return nil, err
	}

	if !utils.CheckPassword(input.Password, user.Password) {
		return nil, errors.New("e-mail ou mot de passe incorrect")
	}

	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshTokenRecord := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(168 * time.Hour),
	}
	db.Create(&refreshTokenRecord)

	user.Password = ""

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func (s *AuthService) VerifyEmail(token string) error {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email_verify_token = ?", token).First(&user).Error; err != nil {
		return errors.New("token invalide ou expiré")
	}

	user.IsEmailVerified = true
	user.EmailVerifyToken = ""

	return db.Save(&user).Error
}

func (s *AuthService) RequestPasswordReset(email string) error {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil
	}

	resetToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	user.ResetPasswordToken = resetToken
	user.ResetPasswordExp = &expiresAt

	if err := db.Save(&user).Error; err != nil {
		return err
	}

	go s.emailService.SendPasswordResetEmail(user.Email, resetToken)

	return nil
}

func (s *AuthService) ResetPassword(token, newPassword string) error {
	db := database.GetDB()

	var user models.User
	if err := db.Where("reset_password_token = ?", token).First(&user).Error; err != nil {
		return errors.New("token invalide ou expiré")
	}

	if user.ResetPasswordExp == nil || time.Now().After(*user.ResetPasswordExp) {
		return errors.New("token expiré")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	user.ResetPasswordToken = ""
	user.ResetPasswordExp = nil

	return db.Save(&user).Error
}

func (s *AuthService) GetGoogleLoginURL(state string) string {
	return s.googleConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

func (s *AuthService) GoogleCallback(code string) (*AuthResponse, error) {
	db := database.GetDB()

	token, err := s.googleConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}

	client := s.googleConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, err
	}

	var user models.User
	result := db.Where("google_id = ?", googleUser.ID).Or("email = ?", googleUser.Email).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			user = models.User{
				Email:           googleUser.Email,
				FirstName:       googleUser.GivenName,
				LastName:        googleUser.FamilyName,
				GoogleID:        googleUser.ID,
				IsEmailVerified: googleUser.VerifiedEmail,
				Provider:        "google",
				Password:        uuid.New().String(),
			}

			if err := db.Create(&user).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	} else {
		if user.GoogleID == "" {
			user.GoogleID = googleUser.ID
			user.Provider = "google"
			db.Save(&user)
		}
	}

	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshTokenRecord := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(168 * time.Hour),
	}
	db.Create(&refreshTokenRecord)

	user.Password = ""

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func (s *AuthService) RefreshAccessToken(refreshToken string) (*AuthResponse, error) {
	db := database.GetDB()

	claims, err := utils.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("token invalide")
	}

	var tokenRecord models.RefreshToken
	if err := db.Where("token = ? AND user_id = ?", refreshToken, claims.UserID).First(&tokenRecord).Error; err != nil {
		return nil, errors.New("token invalide")
	}

	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, errors.New("token expiré")
	}

	var user models.User
	if err := db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, err
	}

	newAccessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	user.Password = ""

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

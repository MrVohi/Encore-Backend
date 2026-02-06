package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	Username       string `json:"username" binding:"required"`
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
	if secret == "" {
		return false, fmt.Errorf("RECAPTCHA_SECRET not set")
	}
	if strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("recaptcha token missing")
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Success    bool     `json:"success"`
		Hostname   string   `json:"hostname"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if !result.Success {
		return false, fmt.Errorf("recaptcha failed: %v", result.ErrorCodes)
	}

	return true, nil
}

func sanitizeUsername(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return ""
	}
	if len(out) > 24 {
		out = out[:24]
	}
	return out
}

func baseFromEmail(email string) string {
	local := email
	if i := strings.Index(email, "@"); i > 0 {
		local = email[:i]
	}
	return sanitizeUsername(local)
}

func usernameAvailable(db *gorm.DB, username string) (bool, error) {
	var u models.User
	err := db.Select("id").
		Where("username = ?", username).
		First(&u).Error

	if err == nil {
		return false, nil
	}
	if err == gorm.ErrRecordNotFound {
		return true, nil
	}
	return false, err
}

func generateUniqueUsername(db *gorm.DB, input RegisterInput) (string, error) {
	base := sanitizeUsername(input.Username)
	if base == "" {
		base = baseFromEmail(input.Email)
	}
	if base == "" {
		base = "user"
	}

	ok, err := usernameAvailable(db, base)
	if err != nil {
		return "", err
	}
	if ok {
		return base, nil
	}

	for i := 2; i <= 9999; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)

		if len(candidate) > 32 {
			suffix := fmt.Sprintf("-%d", i)
			maxBase := 32 - len(suffix)
			if maxBase < 1 {
				candidate = "user" + suffix
			} else if len(base) > maxBase {
				candidate = base[:maxBase] + suffix
			}
		}

		ok, err := usernameAvailable(db, candidate)
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not generate a unique username")
}

func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
	db := database.GetDB().Debug()

	isHuman, err := s.VerifyRecaptcha(input.RecaptchaToken)
	if err != nil || !isHuman {
		return nil, fmt.Errorf("recaptcha: %v", err)
	}

	var existingUser models.User
	result := db.Where("email = ?", input.Email).First(&existingUser)
	if result.Error == nil {
		return nil, errors.New("this email is already in use")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	verifyToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	username, err := generateUniqueUsername(db, input)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Email:            input.Email,
		PasswordHash:     hashedPassword,
		Name:             input.FirstName + " " + input.LastName,
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		Provider:         "local",
		Username:         username,
		IsEmailVerified:  false,
		EmailVerifyToken: verifyToken,
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

	user.PasswordHash = ""

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
			return nil, errors.New("email or password incorrect")
		}
		return nil, err
	}

	if !utils.CheckPassword(input.Password, user.PasswordHash) {
		return nil, errors.New("email or password incorrect")
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

	user.PasswordHash = ""

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
		return errors.New("invalid or expired token")
	}

	user.IsEmailVerified = true
	user.EmailVerifyToken = ""

	return db.Save(&user).Error
}

func (s *AuthService) ResendVerification(email string) error {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil
	}

	if user.IsEmailVerified {
		return nil
	}

	verifyToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		return err
	}

	user.EmailVerifyToken = verifyToken
	if err := db.Save(&user).Error; err != nil {
		return err
	}

	go s.emailService.SendVerificationEmail(user.Email, verifyToken)

	return nil
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
		return errors.New("invalid or expired token")
	}

	if user.ResetPasswordExp == nil || time.Now().After(*user.ResetPasswordExp) {
		return errors.New("token expired")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hashedPassword
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
			// googleUser.ID is a string; models.User.GoogleID is *string, store a pointer
			gid := googleUser.ID
			username, err := generateUniqueUsername(db, RegisterInput{
				Email:    googleUser.Email,
				Username: googleUser.GivenName,
			})
			if err != nil {
				return nil, err
			}

			fullName := strings.TrimSpace(googleUser.GivenName + " " + googleUser.FamilyName)
			if fullName == "" {
				fullName = googleUser.Email
			}

			user = models.User{
				Email:           googleUser.Email,
				Name:            fullName,
				Username:        username,
				FirstName:       googleUser.GivenName,
				LastName:        googleUser.FamilyName,
				GoogleID:        &gid,
				IsEmailVerified: googleUser.VerifiedEmail,
				Provider:        "google",
				PasswordHash:    uuid.New().String(),
			}

			if err := db.Create(&user).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	} else {
		// user.GoogleID is a *string; handle nil or empty
		if user.GoogleID == nil || *user.GoogleID == "" {
			gid := googleUser.ID
			user.GoogleID = &gid
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

	user.PasswordHash = ""

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
		return nil, errors.New("invalid token")
	}

	var tokenRecord models.RefreshToken
	if err := db.Where("token = ? AND user_id = ?", refreshToken, claims.UserID).First(&tokenRecord).Error; err != nil {
		return nil, errors.New("invalid token")
	}

	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	var user models.User
	if err := db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, err
	}

	newAccessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                 string         `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Email              string         `gorm:"uniqueIndex;not null" json:"email"`
	Password           string         `gorm:"not null" json:"-"`
	FirstName          string         `json:"first_name"`
	LastName           string         `json:"last_name"`
	IsEmailVerified    bool           `gorm:"default:false" json:"is_email_verified"`
	EmailVerifyToken   string         `json:"-"`
	ResetPasswordToken string         `json:"-"`
	ResetPasswordExp   *time.Time     `json:"-"`
	GoogleID           string         `gorm:"uniqueIndex" json:"google_id,omitempty"`
	Provider           string         `gorm:"default:'local'" json:"provider"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

type RefreshToken struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    string    `gorm:"not null;index"`
	Token     string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

package database

import (
	"groupie-tracker/internal/authentification/models"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_URL")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL or POSTGRES_URL is required")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")

	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	if os.Getenv("APP_ENV") == "dev" {
		err = DB.AutoMigrate(&models.User{}, &models.RefreshToken{})
		if err != nil {
			log.Fatal("Failed to migrate database:", err)
		}
		log.Println("Database migrated successfully")
	}

	DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower ON users (LOWER(username))")
}

func GetDB() *gorm.DB {
	return DB
}

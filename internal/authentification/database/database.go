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
	dsn := os.Getenv("POSTGRES_URL")

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")

	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	err = DB.AutoMigrate(&models.User{}, &models.RefreshToken{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower ON users (LOWER(username))")

	log.Println("Database migrated successfully")
}

func GetDB() *gorm.DB {
	return DB
}

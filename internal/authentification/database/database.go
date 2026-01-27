package database

import (
	"log"
	"os"

	"groupie-tracker/internal/authentification/models"

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

	log.Println("Database migrated successfully")
}

func GetDB() *gorm.DB {
	return DB
}

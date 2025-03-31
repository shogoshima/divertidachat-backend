package services

import (
	"fmt"
	"os"

	"github.com/shogoshima/divertidachat-backend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() error {
	// Use this format for PostgreSQL connection
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	return err
}

func InitializeDB() error {
	// Initialize the database connection
	if err := ConnectDB(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Chat{},
		&models.ChatUser{},
		&models.Message{},
	); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

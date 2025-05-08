package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect подключается к базе данных PostgreSQL с помощью GORM
func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Connected to database with GORM")

	// Автоматическая миграция схем (создаст таблицы, если их нет)
	// err = db.AutoMigrate(&models.Pin{}, &models.User{}, &models.Action{}, &models.UserSubscription{}, &models.Comment{})
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to auto migrate: %w", err)
	// }

	return db, nil
}

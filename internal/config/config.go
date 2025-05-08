package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv" // Убедимся, что импорт присутствует
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string // Добавляем поле для JWT Secret
}

func LoadConfig() (Config, error) {
	// Загрузка переменных окружения из файла .env (если есть)
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("error loading .env file: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Значение по умолчанию
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET environment variable not set")
	}

	return Config{
		Port:        port,
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret, // Загружаем JWT Secret
	}, nil
}

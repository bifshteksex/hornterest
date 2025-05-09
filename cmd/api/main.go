package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"pornterest/internal/config"
	"pornterest/internal/database"
	"pornterest/internal/handlers"
	"pornterest/internal/routes"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// Загрузка переменных окружения
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключение к базе данных (теперь возвращает *gorm.DB)
	dbGORM, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := dbGORM.DB() // Получаем базовый *sql.DB для корректного defer Close()
	if err != nil {
		log.Fatalf("Failed to get generic database object: %v", err)
	}
	defer sqlDB.Close()

	// Создание обработчиков (передаем *gorm.DB)
	pinHandler := handlers.NewPinHandler(dbGORM)
	userHandler := handlers.NewUserHandler(dbGORM, cfg)
	actionHandler := handlers.NewActionHandler(dbGORM)
	subscriptionHandler := handlers.NewSubscriptionHandler(dbGORM)
	tagHandler := handlers.NewTagHandler(dbGORM)

	// Создаем роутер gorilla/mux
	router := mux.NewRouter()

	// Настройка CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // В production следует указать конкретные домены
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	// Регистрация маршрутов из отдельных файлов
	routes.SetupPinRoutes(router, pinHandler, actionHandler, cfg)
	routes.SetupUserRoutes(router, userHandler, subscriptionHandler, cfg)
	routes.SetupTagRoutes(router, tagHandler, cfg)

	// Маршрут для статики
	router.PathPrefix("/upload/").Handler(http.StripPrefix("/upload/", http.FileServer(http.Dir("./upload"))))

	// Применяем CORS middleware ко всему роутеру
	handler := c.Handler(router)

	// Запуск сервера
	fmt.Printf("Server listening on port %s\n", cfg.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), handler)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

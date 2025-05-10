package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"pornterest/internal/config"
	"pornterest/internal/database"
	"pornterest/internal/elasticsearch"
	"pornterest/internal/handlers"
	"pornterest/internal/routes"
	"pornterest/internal/tasks" // добавляем импорт
	"pornterest/internal/tools"

	"log/slog" // добавляем для логгера

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

	// Настройка логгера
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключение к базе данных
	dbGORM, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := dbGORM.DB()
	if err != nil {
		log.Fatalf("Failed to get generic database object: %v", err)
	}
	defer sqlDB.Close()

	// Инициализация TaskQueue
	taskQueue := tasks.NewTaskQueue(dbGORM, logger)
	taskQueue.StartProcessing()

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewESClient([]string{os.Getenv("ELASTICSEARCH_URL")})
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch client: %v", err)
	}

	// Create index if it doesn't exist
	if err := esClient.CreateIndex(context.Background(), "pins"); err != nil {
		log.Fatalf("Failed to create Elasticsearch index: %v", err)
	}

	// Создание обработчиков
	pinHandler := handlers.NewPinHandler(dbGORM, taskQueue, esClient)
	userHandler := handlers.NewUserHandler(dbGORM, cfg)
	actionHandler := handlers.NewActionHandler(dbGORM)
	subscriptionHandler := handlers.NewSubscriptionHandler(dbGORM)
	tagHandler := handlers.NewTagHandler(dbGORM)

	// Создаем роутер gorilla/mux
	router := mux.NewRouter()

	// Настройка CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	// Регистрация маршрутов
	routes.SetupPinRoutes(router, pinHandler, actionHandler, cfg)
	routes.SetupUserRoutes(router, userHandler, subscriptionHandler, cfg)
	routes.SetupTagRoutes(router, tagHandler, cfg)

	// Маршрут для статики
	router.PathPrefix("/upload/").Handler(http.StripPrefix("/upload/", http.FileServer(http.Dir("./upload"))))

	// Применяем CORS middleware
	handler := c.Handler(router)

	if err := tools.ReindexAllPins(dbGORM, esClient); err != nil {
		log.Printf("Failed to reindex pins: %v", err)
	}

	// Запуск сервера
	fmt.Printf("Server listening on port %s\n", cfg.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), handler)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

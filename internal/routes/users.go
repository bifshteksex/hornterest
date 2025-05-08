package routes

import (
	"fmt"
	"net/http"
	"pornterest/internal/config"
	"pornterest/internal/handlers"
	"pornterest/internal/middleware"

	"github.com/gorilla/mux"
)

// protectedHandler - пример защищенного обработчика (пока оставим здесь)
func protectedHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserID).(int)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("This is a protected resource for user ID: %d!", userID)))
}

// SetupUserRoutes регистрирует маршруты, связанные с пользователями
func SetupUserRoutes(router *mux.Router, userHandler *handlers.UserHandler, subscriptionHandler *handlers.SubscriptionHandler, cfg config.Config) {
	router.HandleFunc("/api/register", userHandler.Register).Methods("POST")
	router.HandleFunc("/api/login", userHandler.Login).Methods("POST")
	router.Handle("/api/protected", middleware.AuthMiddleware(cfg)(http.HandlerFunc(protectedHandler))).Methods("GET")
	router.HandleFunc("/api/users/{id:[0-9]+}", userHandler.GetUserByID).Methods("GET")
	router.Handle("/api/users/{id:[0-9]+}", middleware.AuthMiddleware(cfg)(http.HandlerFunc(userHandler.UpdateUser))).Methods("PUT")
	router.HandleFunc("/api/users/{username}", userHandler.GetUserByUsername).Methods("GET")
	router.HandleFunc("/api/users/{username}/pins", userHandler.GetUserPinsByUsername).Methods("GET")
	router.HandleFunc("/api/users/{username}/saved", userHandler.GetUserSavedPinsByUsername).Methods("GET")

	// Маршруты для подписок
	router.Handle("/api/users/{target_user_id:[0-9]+}/subscribe", middleware.AuthMiddleware(cfg)(http.HandlerFunc(subscriptionHandler.SubscribeUser))).Methods("POST")
	router.Handle("/api/users/{target_user_id:[0-9]+}/unsubscribe", middleware.AuthMiddleware(cfg)(http.HandlerFunc(subscriptionHandler.UnsubscribeUser))).Methods("DELETE")
	router.Handle("/api/users/{target_user_id:[0-9]+}/subscribed", middleware.AuthMiddleware(cfg)(http.HandlerFunc(subscriptionHandler.CheckIfSubscribed))).Methods("GET")
}

package routes

import (
	"net/http"
	"pornterest/internal/config"
	"pornterest/internal/handlers"
	"pornterest/internal/middleware"

	"github.com/gorilla/mux"
)

// SetupPinRoutes регистрирует маршруты, связанные с пинами
func SetupPinRoutes(router *mux.Router, pinHandler *handlers.PinHandler, actionHandler *handlers.ActionHandler, cfg config.Config) {
	router.HandleFunc("/api/pins", pinHandler.GetPins).Methods("GET")
	router.HandleFunc("/api/pins/{id:[0-9]+}", pinHandler.GetPin).Methods("GET")
	router.Handle("/api/pin/upload", middleware.AuthMiddleware(cfg)(http.HandlerFunc(pinHandler.UploadPin))).Methods("POST")

	// Маршруты для лайков
	router.Handle("/api/pins/{id:[0-9]+}/like", middleware.AuthMiddleware(cfg)(http.HandlerFunc(actionHandler.LikePin))).Methods("POST")
	router.Handle("/api/pins/{id:[0-9]+}/unlike", middleware.AuthMiddleware(cfg)(http.HandlerFunc(actionHandler.UnlikePin))).Methods("DELETE")
	router.HandleFunc("/api/pins/{id:[0-9]+}/likes/count", actionHandler.GetPinLikesCount).Methods("GET")
	router.Handle("/api/pins/{id:[0-9]+}/liked", middleware.AuthMiddleware(cfg)(http.HandlerFunc(actionHandler.CheckIfLiked))).Methods("GET")

	// Маршруты для сохранения
	router.Handle("/api/pins/{id:[0-9]+}/save", middleware.AuthMiddleware(cfg)(http.HandlerFunc(actionHandler.SavePin))).Methods("POST")
	router.Handle("/api/pins/{id:[0-9]+}/unsave", middleware.AuthMiddleware(cfg)(http.HandlerFunc(actionHandler.UnsavePin))).Methods("DELETE")
	router.Handle("/api/pins/{id:[0-9]+}/saved", middleware.AuthMiddleware(cfg)(http.HandlerFunc(actionHandler.CheckIfSaved))).Methods("GET")

	// Маршруты для комментариев
	router.Handle("/api/pins/{id:[0-9]+}/comments", middleware.AuthMiddleware(cfg)(http.HandlerFunc(pinHandler.AddComment))).Methods("POST")
	router.HandleFunc("/api/pins/{id:[0-9]+}/comments", pinHandler.GetPinComments).Methods("GET")

	// Поиск пинов
	router.HandleFunc("/api/search", pinHandler.SearchPins)
}

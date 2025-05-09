package routes

import (
	"pornterest/internal/config"
	"pornterest/internal/handlers"

	"github.com/gorilla/mux"
)

func SetupTagRoutes(router *mux.Router, tagHandler *handlers.TagHandler, cfg config.Config) {
	router.HandleFunc("/api/tags/process", tagHandler.ProcessTags).Methods("POST")
	router.HandleFunc("/api/tags", tagHandler.GetAllTags).Methods("GET")
	router.HandleFunc("/api/tags", tagHandler.UpdateTag).Methods("PUT")
}

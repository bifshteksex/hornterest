package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"pornterest/internal/middleware"
	"pornterest/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type ActionHandler struct {
	db *gorm.DB
}

func NewActionHandler(db *gorm.DB) *ActionHandler {
	return &ActionHandler{db: db}
}

// LikePin обрабатывает HTTP POST запрос для лайка пина
func (h *ActionHandler) LikePin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	action := models.UserAction{
		UserID:    userID,
		PinID:     pinID,
		Action:    "like",
		CreatedAt: time.Now(),
	}

	result := h.db.Create(&action)
	if result.Error != nil {
		log.Printf("Failed to like pin %d by user %d: %v", pinID, userID, result.Error)
		http.Error(w, "Failed to like pin", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Pin liked successfully"})
}

// UnlikePin обрабатывает HTTP DELETE запрос для удаления лайка с пина
func (h *ActionHandler) UnlikePin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	result := h.db.Where("user_id = ? AND pin_id = ? AND action = ?", userID, pinID, "like").Delete(&models.UserAction{})
	if result.Error != nil {
		log.Printf("Failed to unlike pin %d by user %d: %v", pinID, userID, result.Error)
		http.Error(w, "Failed to unlike pin", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Like not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Pin unliked successfully"})
}

// GetPinLikesCount обрабатывает HTTP GET запрос для получения количества лайков пина
func (h *ActionHandler) GetPinLikesCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	var count int64
	result := h.db.Model(&models.UserAction{}).Where("pin_id = ? AND action = ?", pinID, "like").Count(&count)
	if result.Error != nil {
		log.Printf("Failed to get likes count for pin %d: %v", pinID, result.Error)
		http.Error(w, "Failed to get likes count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"likes_count": count})
}

// CheckIfLiked обрабатывает HTTP GET запрос для проверки, лайкнул ли пользователь пин
func (h *ActionHandler) CheckIfLiked(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	var exists bool
	result := h.db.Where("user_id = ? AND pin_id = ? AND action = ?", userID, pinID, "like").First(&models.UserAction{})
	if result.Error == nil {
		exists = true
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) { // Исправленная строка
		log.Printf("Failed to check if user %d liked pin %d: %v", userID, pinID, result.Error)
		http.Error(w, "Failed to check if liked", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"liked": exists})
}

// SavePin обрабатывает HTTP POST запрос для сохранения пина
func (h *ActionHandler) SavePin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	action := models.UserAction{
		UserID:    userID,
		PinID:     pinID,
		Action:    "save",
		CreatedAt: time.Now(),
	}

	result := h.db.Create(&action)
	if result.Error != nil {
		log.Printf("Failed to save pin %d by user %d: %v", pinID, userID, result.Error)
		http.Error(w, "Failed to save pin", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Pin saved successfully"})
}

// UnsavePin обрабатывает HTTP DELETE запрос для отмены сохранения пина
func (h *ActionHandler) UnsavePin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	result := h.db.Where("user_id = ? AND pin_id = ? AND action = ?", userID, pinID, "save").Delete(&models.UserAction{})
	if result.Error != nil {
		log.Printf("Failed to unsave pin %d by user %d: %v", pinID, userID, result.Error)
		http.Error(w, "Failed to unsave pin", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Save not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Pin unsaved successfully"})
}

// CheckIfSaved обрабатывает HTTP GET запрос для проверки, сохранен ли пин пользователем
func (h *ActionHandler) CheckIfSaved(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	var exists bool
	result := h.db.Where("user_id = ? AND pin_id = ? AND action = ?", userID, pinID, "save").First(&models.UserAction{})
	if result.Error == nil {
		exists = true
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) { // Исправленная строка
		log.Printf("Failed to check if user %d saved pin %d: %v", userID, pinID, result.Error)
		http.Error(w, "Failed to check if saved", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"saved": exists})
}

// extractPinID извлекает ID пина из URL (теперь использует gorilla/mux vars)
func (h *ActionHandler) extractPinID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	pinIDStr, ok := vars["id"]
	if !ok {
		return 0, fmt.Errorf("pin ID is missing in parameters")
	}
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pin ID: %w", err)
	}
	return pinID, nil
}

// extractPinIDFromCountRoute извлекает ID пина из URL для маршрутов с count (теперь использует gorilla/mux vars)
func (h *ActionHandler) extractPinIDFromCountRoute(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	pinIDStr, ok := vars["id"]
	if !ok {
		return 0, fmt.Errorf("pin ID is missing in parameters")
	}
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pin ID: %w", err)
	}
	return pinID, nil
}

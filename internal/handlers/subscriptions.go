package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"pornterest/internal/middleware"
	"pornterest/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// SubscriptionHandler обрабатывает запросы, связанные с подписками пользователей
type SubscriptionHandler struct {
	db *gorm.DB
}

// NewSubscriptionHandler создает новый экземпляр SubscriptionHandler
func NewSubscriptionHandler(db *gorm.DB) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

// SubscribeUser обрабатывает HTTP POST запрос для подписки одного пользователя на другого
func (h *SubscriptionHandler) SubscribeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)
	vars := mux.Vars(r)
	targetUserIDStr, ok := vars["target_user_id"]
	if !ok {
		http.Error(w, "Target user ID is required", http.StatusBadRequest)
		return
	}

	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		http.Error(w, "Invalid target user ID", http.StatusBadRequest)
		return
	}

	if userID == targetUserID {
		http.Error(w, "Cannot subscribe to yourself", http.StatusBadRequest)
		return
	}

	subscription := models.UserSubscription{
		UserID:       userID,
		TargetUserID: targetUserID,
		CreatedAt:    time.Now(),
	}

	result := h.db.Create(&subscription)
	if result.Error != nil {
		log.Printf("Failed to subscribe user %d to %d: %v", userID, targetUserID, result.Error)
		http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Subscribed successfully"})
}

// UnsubscribeUser обрабатывает HTTP DELETE запрос для отписки одного пользователя от другого
func (h *SubscriptionHandler) UnsubscribeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)
	vars := mux.Vars(r)
	targetUserIDStr, ok := vars["target_user_id"]
	if !ok {
		http.Error(w, "Target user ID is required", http.StatusBadRequest)
		return
	}

	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		http.Error(w, "Invalid target user ID", http.StatusBadRequest)
		return
	}

	result := h.db.Where("user_id = ? AND target_user_id = ?", userID, targetUserID).Delete(&models.UserSubscription{})
	if result.Error != nil {
		log.Printf("Failed to unsubscribe user %d from %d: %v", userID, targetUserID, result.Error)
		http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Unsubscribed successfully"})
}

// CheckIfSubscribed обрабатывает HTTP GET запрос для проверки, подписан ли текущий пользователь на указанного
func (h *SubscriptionHandler) CheckIfSubscribed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)
	vars := mux.Vars(r)
	targetUserIDStr, ok := vars["target_user_id"]
	if !ok {
		http.Error(w, "Target user ID is required", http.StatusBadRequest)
		return
	}

	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		http.Error(w, "Invalid target user ID", http.StatusBadRequest)
		return
	}

	var exists bool
	result := h.db.Where("user_id = ? AND target_user_id = ?", userID, targetUserID).First(&models.UserSubscription{})
	if result.Error == nil {
		exists = true
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Printf("Failed to check if user %d is subscribed to %d: %v", userID, targetUserID, result.Error)
		http.Error(w, "Failed to check subscription status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"subscribed": exists})
}

// GetUserFollowersCount обрабатывает HTTP GET запрос для получения количества подписчиков пользователя
func (h *SubscriptionHandler) GetUserFollowersCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	userIDStr, ok := vars["user_id"]
	if !ok {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var count int64
	result := h.db.Model(&models.UserSubscription{}).Where("target_user_id = ?", userID).Count(&count)
	if result.Error != nil {
		log.Printf("Failed to get followers count for user %d: %v", userID, result.Error)
		http.Error(w, "Failed to get followers count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"followers_count": count})
}

// GetUserFollowingCount обрабатывает HTTP GET запрос для получения количества подписок пользователя
func (h *SubscriptionHandler) GetUserFollowingCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserID).(int)

	var count int64
	result := h.db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		log.Printf("Failed to get following count for user %d: %v", userID, result.Error)
		http.Error(w, "Failed to get following count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"following_count": count})
}

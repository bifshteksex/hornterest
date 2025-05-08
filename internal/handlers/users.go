package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"pornterest/internal/config" // Импортируем пакет config
	"pornterest/internal/models"

	"github.com/golang-jwt/jwt/v5" // Импортируем библиотеку JWT
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserHandler обрабатывает запросы, связанные с пользователями
type UserHandler struct {
	db     *gorm.DB
	config config.Config // Добавляем поле для хранения конфигурации
}

// NewUserHandler создает новый экземпляр UserHandler, принимая конфигурацию
func NewUserHandler(db *gorm.DB, cfg config.Config) *UserHandler {
	return &UserHandler{db: db, config: cfg}
}

// Register обрабатывает HTTP POST запрос для регистрации нового пользователя
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Простая валидация обязательных полей
	if user.Nickname == "" || user.Email == "" || user.Password == "" {
		http.Error(w, "Nickname, email, and password are required", http.StatusBadRequest)
		return
	}

	// Хеширование пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword) // Сохраняем хеш в объект пользователя

	// Установка времени создания и обновления
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Сохранение пользователя в базе данных
	result := h.db.Create(&user)
	if result.Error != nil {
		log.Printf("Failed to create user: %v", result.Error)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result.RowsAffected); err != nil { // GORM возвращает RowsAffected при Create
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Login обрабатывает HTTP POST запрос для авторизации пользователя
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		Identifier string `json:"identifier"` // Может быть никнеймом или email
		Password   string `json:"passwordLogin"`
	}

	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if credentials.Identifier == "" || credentials.Password == "" {
		http.Error(w, "Identifier and password are required", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по никнейму или email
	var user models.User
	result := h.db.Where("nickname = ? OR email = ?", credentials.Identifier, credentials.Identifier).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		log.Printf("Failed to get user: %v", result.Error)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Сравниваем введенный пароль с хешированным паролем из базы данных
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Генерация JWT
	token, err := h.generateJWT(user.ID)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token":    token,
		"id":       user.ID,
		"nickname": user.Nickname,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateJWT генерирует новый JWT для указанного ID пользователя
func (h *UserHandler) generateJWT(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Токен действителен в течение 24 часов
		"iat":     time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.config.JWTSecret)) // Используем JWTSecret из конфигурации
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// UpdateUser обрабатывает HTTP PUT запрос для обновления информации о пользователе.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	userIDStr, ok := vars["id"]
	if !ok {
		http.Error(w, "User ID parameter is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var updatedUser models.User
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Проверяем, что ID из URL совпадает с ID в теле запроса (если он есть)
	if updatedUser.ID != 0 && updatedUser.ID != userID {
		http.Error(w, "User ID in path does not match ID in body", http.StatusBadRequest)
		return
	}
	updatedUser.ID = userID // Устанавливаем ID из URL

	// Обновляем только разрешенные поля
	var userToUpdate models.User
	result := h.db.First(&userToUpdate, userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to find user for update: %v", result.Error)
		http.Error(w, "Failed to find user", http.StatusInternalServerError)
		return
	}

	userToUpdate.UpdatedAt = time.Now()

	if updatedUser.Nickname != "" {
		userToUpdate.Nickname = updatedUser.Nickname
	}
	if updatedUser.Description != "" {
		userToUpdate.Description = updatedUser.Description
	}
	userToUpdate.Hidden = updatedUser.Hidden
	userToUpdate.Private = updatedUser.Private
	userToUpdate.Verification = updatedUser.Verification
	if updatedUser.Name != "" {
		userToUpdate.Name = updatedUser.Name
	}
	if updatedUser.Surname != "" {
		userToUpdate.Surname = updatedUser.Surname
	}
	if updatedUser.Birth != nil {
		userToUpdate.Birth = updatedUser.Birth
	}
	if updatedUser.Sex != "" {
		userToUpdate.Sex = updatedUser.Sex
	}
	if updatedUser.Country != nil {
		if *updatedUser.Country != "" {
			userToUpdate.Country = updatedUser.Country
		}
	}

	userToUpdate.Comment = updatedUser.Comment
	userToUpdate.Autoplay = updatedUser.Autoplay
	userToUpdate.TwoFa = updatedUser.TwoFa

	updateResult := h.db.Save(&userToUpdate)
	if updateResult.Error != nil {
		log.Printf("Failed to update user: %v", updateResult.Error)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "User with ID %d updated successfully", userID)
}

// GetUserByID обрабатывает HTTP GET запрос для получения информации о пользователе по ID
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	userIDStr, ok := vars["id"]
	if !ok {
		http.Error(w, "User ID parameter is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	result := h.db.First(&user, userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get user by ID: %v", result.Error)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var followersCount int64
	h.db.Model(&models.UserSubscription{}).Where("target_user_id = ?", user.ID).Count(&followersCount)

	var followingCount int64
	h.db.Model(&models.UserSubscription{}).Where("user_id = ?", user.ID).Count(&followingCount)

	userResponse := struct {
		ID           int        `json:"id"`
		Nickname     string     `json:"nickname"`
		Description  string     `json:"description"`
		Hidden       bool       `json:"hidden"`
		Private      bool       `json:"private"`
		Verification bool       `json:"verification"`
		Name         string     `json:"name"`
		Surname      string     `json:"surname"`
		Birth        *time.Time `json:"birth"`
		Sex          string     `json:"sex"`
		Country      *string    `json:"country"`
		Lang         *string    `json:"lang"`
		Mentions     *string    `json:"mentions"`
		Comment      bool       `json:"comment"`
		Autoplay     bool       `json:"autoplay"`
		TwoFa        bool       `json:"2fa"`
		Email        string     `json:"email"`
		CreatedAt    time.Time  `json:"created_at"`
		UpdatedAt    time.Time  `json:"updated_at"`
		Followers    int64      `json:"followers"`
		Following    int64      `json:"following"`
	}{
		ID:           user.ID,
		Nickname:     user.Nickname,
		Description:  user.Description,
		Hidden:       user.Hidden,
		Private:      user.Private,
		Verification: user.Verification,
		Name:         user.Name,
		Surname:      user.Surname,
		Birth:        user.Birth,
		Sex:          user.Sex,
		Country:      user.Country,
		Lang:         user.Lang,
		Mentions:     user.Mentions,
		Comment:      user.Comment,
		Autoplay:     user.Autoplay,
		TwoFa:        user.TwoFa,
		Email:        user.Email,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Followers:    followersCount,
		Following:    followingCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userResponse); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetUserByUsername обрабатывает HTTP GET запрос для получения информации о пользователе по никнейму
func (h *UserHandler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	username := vars["username"]
	if username == "" {
		http.Error(w, "Username parameter is required", http.StatusBadRequest)
		return
	}

	var user models.User
	result := h.db.Where("nickname = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get user by username: %v", result.Error)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var followersCount int64
	h.db.Model(&models.UserSubscription{}).Where("target_user_id = ?", user.ID).Count(&followersCount)

	var followingCount int64
	h.db.Model(&models.UserSubscription{}).Where("user_id = ?", user.ID).Count(&followingCount)

	userResponse := struct {
		ID           int        `json:"id"`
		Nickname     string     `json:"nickname"`
		Description  string     `json:"description"`
		Hidden       bool       `json:"hidden"`
		Private      bool       `json:"private"`
		Verification bool       `json:"verification"`
		Name         string     `json:"name"`
		Surname      string     `json:"surname"`
		Birth        *time.Time `json:"birth"`
		Sex          string     `json:"sex"`
		Country      *string    `json:"country"`
		Lang         *string    `json:"lang"`
		Mentions     *string    `json:"mentions"`
		Comment      bool       `json:"comment"`
		Autoplay     bool       `json:"autoplay"`
		TwoFa        bool       `json:"2fa"`
		Email        string     `json:"email"`
		CreatedAt    time.Time  `json:"created_at"`
		UpdatedAt    time.Time  `json:"updated_at"`
		Followers    int64      `json:"followers"`
		Following    int64      `json:"following"`
	}{
		ID:           user.ID,
		Nickname:     user.Nickname,
		Description:  user.Description,
		Hidden:       user.Hidden,
		Private:      user.Private,
		Verification: user.Verification,
		Name:         user.Name,
		Surname:      user.Surname,
		Birth:        user.Birth,
		Sex:          user.Sex,
		Country:      user.Country,
		Lang:         user.Lang,
		Mentions:     user.Mentions,
		Comment:      user.Comment,
		Autoplay:     user.Autoplay,
		TwoFa:        user.TwoFa,
		Email:        user.Email,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Followers:    followersCount,
		Following:    followingCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userResponse); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetUserPinsByUsername обрабатывает HTTP GET запрос для получения пинов пользователя по никнейму с пагинацией
func (h *UserHandler) GetUserPinsByUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	username := vars["username"]
	if username == "" {
		http.Error(w, "Username parameter is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit := 20 // Значение по умолчанию
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}

	page := 1 // Значение по умолчанию
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err == nil && p > 0 {
			page = p
		}
	}

	offset := (page - 1) * limit

	var user models.User
	result := h.db.Where("nickname = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get user by username: %v", result.Error)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var pins []models.Pin
	pinsResult := h.db.Where("user_id = ?", user.ID).Limit(limit).Offset(offset).Order("id DESC").Find(&pins)
	if pinsResult.Error != nil {
		log.Printf("Failed to get user pins: %v", pinsResult.Error)
		http.Error(w, "Failed to fetch user pins", http.StatusInternalServerError)
		return
	}

	var totalCount int64
	h.db.Model(&models.Pin{}).Where("user_id = ?", user.ID).Count(&totalCount)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(int(totalCount)))
	if err := json.NewEncoder(w).Encode(pins); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetUserSavedPinsByUsername обрабатывает HTTP GET запрос для получения сохраненных пинов пользователя по никнейму с пагинацией
func (h *UserHandler) GetUserSavedPinsByUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	username := vars["username"]
	if username == "" {
		http.Error(w, "Username parameter is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit := 20 // Значение по умолчанию
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}

	page := 1 // Значение по умолчанию
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err == nil && p > 0 {
			page = p
		}
	}

	offset := (page - 1) * limit

	var user models.User
	result := h.db.Where("nickname = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get user by username: %v", result.Error)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var savedPins []models.Pin
	savedPinsResult := h.db.Joins("JOIN user_actions ON user_actions.pin_id = pins.id").
		Where("user_actions.user_id = ? AND user_actions.action = ?", user.ID, "save").
		Limit(limit).
		Offset(offset).
		Order("user_actions.created_at DESC").
		Find(&savedPins)
	if savedPinsResult.Error != nil {
		log.Printf("Failed to get user saved pins: %v", savedPinsResult.Error)
		http.Error(w, "Failed to fetch user saved pins", http.StatusInternalServerError)
		return
	}

	var totalCount int64
	h.db.Model(&models.UserAction{}).Where("user_id = ? AND action = ?", user.ID, "save").Count(&totalCount)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(int(totalCount)))
	if err := json.NewEncoder(w).Encode(savedPins); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

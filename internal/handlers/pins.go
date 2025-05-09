package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pornterest/internal/middleware"
	"pornterest/internal/models"
	"pornterest/internal/tasks"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// PinHandler обрабатывает запросы, связанные с пинами
type PinHandler struct {
	db        *gorm.DB
	taskQueue *tasks.TaskQueue // используем полное имя с пакетом
}

func NewPinHandler(db *gorm.DB, taskQueue *tasks.TaskQueue) *PinHandler {
	return &PinHandler{
		db:        db,
		taskQueue: taskQueue,
	}
}

// GetPins обрабатывает HTTP GET запрос для получения пинов с пагинацией
func (h *PinHandler) GetPins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit := 10 // Значение по умолчанию
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

	var pins []models.Pin
	result := h.db.Limit(limit).Offset(offset).Order("id DESC").Find(&pins)
	if result.Error != nil {
		http.Error(w, "Failed to fetch pins", http.StatusInternalServerError)
		log.Printf("Failed to fetch pins: %v", result.Error)
		return
	}

	var totalCount int64
	h.db.Model(&models.Pin{}).Count(&totalCount)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(int(totalCount))) // Отправляем общее количество для frontend'а (опционально)
	if err := json.NewEncoder(w).Encode(pins); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetPin обрабатывает HTTP GET запрос для получения информации о пине по ID
func (h *PinHandler) GetPin(w http.ResponseWriter, r *http.Request) {
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

	var pin models.Pin
	result := h.db.First(&pin, pinID)
	if result.Error != nil {
		if gorm.ErrRecordNotFound == result.Error {
			http.Error(w, "Pin not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch pin", http.StatusInternalServerError)
		log.Printf("Failed to fetch pin: %v", result.Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pin); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UploadPin обрабатывает HTTP POST запрос для загрузки нового пина
func (h *PinHandler) UploadPin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем ID пользователя из контекста
	userID := r.Context().Value(middleware.UserID).(int)

	// Максимальный размер загружаемого файла - 10MB
	err := r.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Получаем текстовые поля
	title := r.FormValue("title")
	description := r.FormValue("description")
	original := r.FormValue("link")
	allowCommentsStr := r.FormValue("allowComments")
	isAiGeneratedStr := r.FormValue("isAiGenerated")
	widthStr := r.FormValue("width")
	heightStr := r.FormValue("height")
	durationStr := r.FormValue("duration")

	// Обработка булевых значений
	allowComments := strings.ToLower(allowCommentsStr) == "true"
	isAiGenerated := strings.ToLower(isAiGeneratedStr) == "true"

	// Обработка числовых значений
	var width *int
	if widthStr != "" {
		wVal, err := strconv.Atoi(widthStr)
		if err == nil {
			width = &wVal
		} else {
			log.Printf("Failed to parse width: %v", err)
		}
	}

	var height *int
	if heightStr != "" {
		hVal, err := strconv.Atoi(heightStr)
		if err == nil {
			height = &hVal
		} else {
			log.Printf("Failed to parse height: %v", err)
		}
	}

	var duration *float64
	if durationStr != "" {
		dVal, err := strconv.ParseFloat(durationStr, 64)
		if err == nil {
			duration = &dVal
		} else {
			log.Printf("Failed to parse duration: %v", err)
		}
	}

	// Получаем файл
	file, fileHeader, err := r.FormFile("media")
	if err != nil {
		http.Error(w, "Failed to retrieve file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Создаем директорию /upload, если ее нет
	uploadDir := "upload"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadDir, os.ModeDir|0755)
		if err != nil {
			log.Printf("Failed to create upload directory: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
	}

	// Генерируем уникальное имя файла
	timestamp := time.Now().UnixNano()
	filenameParts := strings.Split(fileHeader.Filename, ".")
	var fileExtension string
	if len(filenameParts) > 1 {
		fileExtension = "." + filenameParts[len(filenameParts)-1]
	}
	newFilename := fmt.Sprintf("%d_%d%s", userID, timestamp, fileExtension)
	filePath := fmt.Sprintf("%s/%s", uploadDir, newFilename)

	// Сохраняем файл на сервере
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file on server: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("Failed to copy file to server: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Определяем тип файла (image, video, gif) на основе Content-Type
	contentType := fileHeader.Header.Get("Content-Type")
	var fileType *string
	if strings.HasPrefix(contentType, "image/") {
		t := "image"
		fileType = &t
	} else if strings.HasPrefix(contentType, "video/") {
		t := "video"
		fileType = &t
	} else if contentType == "image/gif" {
		t := "gif"
		fileType = &t
	}

	// Сохраняем информацию о пине в базе данных
	pin := &models.Pin{
		Path:        "http://localhost:8080/" + filePath,
		Description: description,
		UserID:      userID,
		Original:    &original,
		Comment:     &allowComments,
		Ai:          &isAiGenerated,
		Type:        fileType,
		Title:       title,
		Width:       width,
		Height:      height,
		Duration:    duration,
		CreatedAt:   time.Now(), // GORM может делать это автоматически
		UpdatedAt:   time.Now(), // GORM может делать это автоматически
	}

	result := h.db.Create(pin)
	if result.Error != nil {
		log.Printf("Failed to save pin info to database: %v", result.Error)
		http.Error(w, "Failed to save pin info", http.StatusInternalServerError)
		return
	}

	tagsStr := r.FormValue("tags")
	if tagsStr != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			log.Printf("Failed to parse tags: %v", err)
		} else {
			// Создаем связи между пином и тегами
			for _, tagTitle := range tags {
				var tag models.Tag
				// Ищем существующий тег
				result := h.db.Where("title_model = ?", tagTitle).First(&tag)
				if result.Error == gorm.ErrRecordNotFound {
					// Если тег не найден, создаем новый
					tag = models.Tag{
						TitleModel: tagTitle,
						TitleEN:    formatModelTagToEnglish(tagTitle),
						Count:      0,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}
					if err := h.db.Create(&tag).Error; err != nil {
						log.Printf("Failed to create tag: %v", err)
						continue
					}
				}

				// Создаем связь между пином и тегом
				pinTag := models.PinTag{
					PinID:     pin.ID,
					TagID:     tag.ID,
					CreatedAt: time.Now(),
				}
				if err := h.db.Create(&pinTag).Error; err != nil {
					log.Printf("Failed to create pin-tag relation: %v", err)
					continue
				}

				// Добавляем тег в очередь на перевод только после успешного создания связи
				if tag.TitleRU == "" {
					h.taskQueue.AddTranslationTask(tag.ID)
				}
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Pin uploaded successfully", "path": filePath, "id": fmt.Sprintf("%d", pin.ID)})
}

// AddComment обрабатывает добавление нового комментария к пину
func (h *PinHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pinIDStr := vars["id"]
	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	// Получаем ID пользователя из middleware (предполагается, что он есть)
	userID := r.Context().Value(middleware.UserID).(int)

	var commentPayload struct {
		Content   string `json:"content"`
		ReplyToID *int   `json:"reply_to_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&commentPayload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if commentPayload.Content == "" {
		http.Error(w, "Comment content cannot be empty", http.StatusBadRequest)
		return
	}

	newComment := models.Comment{
		UserID:    userID,
		PinID:     pinID,
		Content:   commentPayload.Content,
		ReplyToID: commentPayload.ReplyToID,
		CreatedAt: time.Now(), // GORM может делать это автоматически
		UpdatedAt: time.Now(), // GORM может делать это автоматически
	}

	result := h.db.Create(&newComment) // Теперь должен быть метод Create от GORM
	if result.Error != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newComment)
}

// GetPinComments обрабатывает HTTP GET запрос для получения комментариев к пину
func (h *PinHandler) GetPinComments(w http.ResponseWriter, r *http.Request) {
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

	var comments []models.Comment
	result := h.db.Where("pin_id = ?", pinID).Order("created_at DESC").Find(&comments)
	if result.Error != nil {
		http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
		log.Printf("Failed to fetch comments: %v", result.Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

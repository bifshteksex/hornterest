package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pornterest/internal/models"

	"gorm.io/gorm"
)

type TagHandler struct {
	db *gorm.DB
}

func NewTagHandler(db *gorm.DB) *TagHandler {
	return &TagHandler{db: db}
}

type ProcessTagsRequest struct {
	Tags []string `json:"tags"`
}

// ProcessTags обрабатывает массив тегов от DeepDanbooru
type ProcessTagsResponse struct {
	Tags []TagInfo `json:"tags"`
}

type TagInfo struct {
	Title string `json:"title"`
	Count int    `json:"count"`
}

// Обновляем функцию ProcessTags
func (h *TagHandler) ProcessTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProcessTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var processedTags []TagInfo

	for _, modelTag := range req.Tags {
		var tag models.Tag

		// Ищем тег в БД
		result := h.db.Where("title_model = ?", modelTag).First(&tag)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// Создаем новый тег
				englishTitle := formatModelTagToEnglish(modelTag)
				tag = models.Tag{
					TitleModel: modelTag,
					TitleEN:    englishTitle,
					Count:      0,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				if err := h.db.Create(&tag).Error; err != nil {
					continue
				}

				processedTags = append(processedTags, TagInfo{
					Title: englishTitle,
					Count: tag.Count,
				})
			} else {
				continue
			}
		} else {
			// Используем русский перевод если есть, иначе английский
			title := tag.TitleEN
			if tag.TitleRU != "" {
				title = tag.TitleRU
			}
			processedTags = append(processedTags, TagInfo{
				Title: title,
				Count: tag.Count,
			})
		}
	}

	response := ProcessTagsResponse{
		Tags: processedTags,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

type GetTagsResponse struct {
	Tags  []models.Tag `json:"tags"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

// GetAllTags возвращает все теги с пагинацией
func (h *TagHandler) GetAllTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	var tags []models.Tag
	var total int64

	if err := h.db.Model(&models.Tag{}).Count(&total).Error; err != nil {
		log.Printf("Failed to count tags: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.db.
		Limit(limit).
		Offset((page - 1) * limit).
		Order("count DESC").
		Find(&tags).Error; err != nil {
		log.Printf("Failed to fetch tags: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := GetTagsResponse{
		Tags:  tags,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

type UpdateTagRequest struct {
	TitleRu string `json:"title_ru"`
}

// UpdateTag обновляет русский перевод тега
func (h *TagHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем ID тега из URL
	tagID := strings.TrimPrefix(r.URL.Path, "/api/tags/")

	var req UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result := h.db.Model(&models.Tag{}).
		Where("id = ?", tagID).
		Updates(map[string]interface{}{
			"title_ru":   req.TitleRu,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Tag updated successfully",
	})
}

// formatModelTagToEnglish преобразует тег из формата модели в английский формат
func formatModelTagToEnglish(modelTag string) string {
	// Добавляем пробел после цифр
	re := regexp.MustCompile(`(\d+)([a-zA-Z])`)
	modelTag = re.ReplaceAllString(modelTag, "$1 $2")

	// Заменяем подчеркивания на пробелы и делаем каждое слово с большой буквы
	words := strings.Split(modelTag, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}

	return strings.Join(words, " ")
}

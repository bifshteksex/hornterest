package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"pornterest/internal/models"

	"gorm.io/gorm"
)

type TaskQueue struct {
	translationTasks chan TranslationTask
	db               *gorm.DB
	logger           *slog.Logger
}

type TranslationTask struct {
	TagID int // Изменил тип с uint на int, чтобы соответствовать models.Tag
}

func NewTaskQueue(db *gorm.DB, logger *slog.Logger) *TaskQueue {
	return &TaskQueue{
		translationTasks: make(chan TranslationTask, 100),
		db:               db,
		logger:           logger,
	}
}

func (t *TaskQueue) AddTranslationTask(tagID int) {
	task := TranslationTask{TagID: tagID}
	select {
	case t.translationTasks <- task:
		t.logger.Info("added translation task", "tag_id", tagID)
	default:
		t.logger.Error("translation queue is full", "tag_id", tagID)
	}
}

func (t *TaskQueue) ProcessTranslationTasks() {
	for task := range t.translationTasks {
		var tag models.Tag
		if err := t.db.First(&tag, task.TagID).Error; err != nil {
			t.logger.Error("failed to get tag",
				"error", err,
				"tag_id", task.TagID,
			)
			continue
		}

		if tag.TitleEN == "" { // Обратите внимание на изменение TitleEn на TitleEN
			t.logger.Error("tag has no English title", "tag_id", task.TagID)
			continue
		}

		translatedText, err := translateText(tag.TitleEN)
		if err != nil {
			t.logger.Error("failed to translate tag",
				"error", err,
				"tag_id", task.TagID,
				"text", tag.TitleEN,
			)
			continue
		}

		tag.TitleRU = translatedText // Обратите внимание на изменение TitleRu на TitleRU
		tag.UpdatedAt = time.Now()
		if err := t.db.Save(&tag).Error; err != nil {
			t.logger.Error("failed to update tag",
				"error", err,
				"tag_id", task.TagID,
			)
			continue
		}

		t.logger.Info("successfully translated tag",
			"tag_id", tag.ID,
			"from", tag.TitleEN,
			"to", tag.TitleRU,
		)

		time.Sleep(time.Second)
	}
}

func translateText(text string) (string, error) {
	const (
		endpoint = "https://translate.api.cloud.yandex.net/translate/v2/translate"
	)

	// Создаем тело запроса
	requestBody := struct {
		TargetLanguageCode string   `json:"targetLanguageCode"`
		Texts              []string `json:"texts"`
		FolderID           string   `json:"folderId"`
	}{
		TargetLanguageCode: "ru",
		Texts:              []string{text},
		FolderID:           os.Getenv("FOLDER_ID_TRANS"),
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Создаем POST запрос
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", os.Getenv("API_KEY_TRANS")))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Структура для ответа от Yandex
	var result struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Translations) == 0 {
		return "", fmt.Errorf("no translations returned")
	}

	return result.Translations[0].Text, nil
}

func (t *TaskQueue) StartProcessing() {
	go t.ProcessTranslationTasks()
}

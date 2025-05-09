package tasks

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
	endpoint := fmt.Sprintf(
		"https://api.mymemory.translated.net/get?q=%s&langpair=en|ru",
		url.QueryEscape(text),
	)

	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ResponseData struct {
			TranslatedText string `json:"translatedText"`
		} `json:"responseData"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ResponseData.TranslatedText, nil
}

func (t *TaskQueue) StartProcessing() {
	go t.ProcessTranslationTasks()
}

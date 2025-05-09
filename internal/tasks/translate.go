package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type TranslationTask struct {
	TagID uint
}

func (t *TaskQueue) AddTranslationTask(tagID uint) {
	task := TranslationTask{TagID: tagID}
	t.translationTasks <- task
}

func (t *TaskQueue) ProcessTranslationTasks() {
	for task := range t.translationTasks {
		tag, err := t.db.GetTagByID(task.TagID)
		if err != nil {
			continue
		}

		// Используем MyMemory API для перевода
		translatedText, err := translateText(tag.TitleEn)
		if err != nil {
			continue
		}

		// Обновляем тег в БД
		tag.TitleRu = translatedText
		t.db.UpdateTag(tag)
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

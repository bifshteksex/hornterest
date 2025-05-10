package tools

import (
	"context"
	"fmt"
	"log"

	"pornterest/internal/elasticsearch"
	"pornterest/internal/models"

	"gorm.io/gorm"
)

func ReindexAllPins(db *gorm.DB, es *elasticsearch.ESClient) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if es == nil {
		return fmt.Errorf("elasticsearch client is nil")
	}

	var pins []models.Pin
	if err := db.Find(&pins).Error; err != nil {
		return fmt.Errorf("failed to fetch pins: %v", err)
	}

	log.Printf("Found %d pins to reindex", len(pins))

	for _, pin := range pins {
		// Создаем копию pin для безопасной передачи указателя
		currentPin := pin

		// Получаем теги для текущего пина
		var tags []models.Tag
		if err := db.Table("tags").
			Joins("JOIN pin_tags ON pin_tags.tag_id = tags.id").
			Where("pin_tags.pin_id = ?", currentPin.ID).
			Find(&tags).Error; err != nil {
			log.Printf("Failed to fetch tags for pin %d: %v", currentPin.ID, err)
			// Продолжаем с пустым списком тегов
			tags = make([]models.Tag, 0)
		}

		log.Printf("Indexing pin %d with %d tags", currentPin.ID, len(tags))

		if err := es.IndexPin(context.Background(), &currentPin, tags); err != nil {
			log.Printf("Failed to index pin %d: %v", currentPin.ID, err)
			continue
		}

		log.Printf("Successfully indexed pin %d", currentPin.ID)
	}

	log.Printf("Reindexing completed")
	return nil
}

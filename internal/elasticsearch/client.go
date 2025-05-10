package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"pornterest/internal/models"

	"github.com/elastic/go-elasticsearch/v8"
)

type ESClient struct {
	client *elasticsearch.Client
}

func NewESClient(addresses []string) (*ESClient, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &ESClient{client: client}, nil
}

func (es *ESClient) CreateIndex(ctx context.Context, indexName string) error {
	res, err := es.client.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}

	if res.StatusCode == 404 {
		res, err = es.client.Indices.Create(
			indexName,
			es.client.Indices.Create.WithBody(strings.NewReader(PinMapping)),
		)
		if err != nil {
			return err
		}
		if res.IsError() {
			return fmt.Errorf("error creating index: %s", res.String())
		}
	}

	return nil
}

type PinDocument struct {
	ID               int           `json:"id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	Original         string        `json:"original,omitempty"`
	OriginalFileName string        `json:"original_file_name,omitempty"`
	Path             string        `json:"path"`
	Type             string        `json:"type"`
	Tags             []TagDocument `json:"tags"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type TagDocument struct {
	TitleEN string `json:"title_en"`
	TitleRU string `json:"title_ru"`
}

func (es *ESClient) IndexPin(ctx context.Context, pin *models.Pin, tags []models.Tag) error {
	if pin == nil {
		return fmt.Errorf("pin cannot be nil")
	}

	pinDoc := PinDocument{
		ID:          pin.ID,
		Title:       pin.Title,
		Description: pin.Description,
		Path:        pin.Path,
		CreatedAt:   pin.CreatedAt,
		UpdatedAt:   pin.UpdatedAt,
	}

	if pin.OriginalFileName != nil {
		pinDoc.OriginalFileName = *pin.OriginalFileName
	}
	if pin.Type != nil {
		pinDoc.Type = *pin.Type
	}

	// Инициализируем пустой слайс тегов, если tags == nil
	pinDoc.Tags = make([]TagDocument, 0)
	if len(tags) > 0 {
		pinDoc.Tags = make([]TagDocument, len(tags))
		for i, tag := range tags {
			pinDoc.Tags[i] = TagDocument{
				TitleEN: tag.TitleEN,
				TitleRU: tag.TitleRU,
			}
		}
	}

	data, err := json.Marshal(pinDoc)
	if err != nil {
		return fmt.Errorf("error marshaling document: %v", err)
	}

	log.Printf("Indexing pin %d with data: %s", pin.ID, string(data))

	res, err := es.client.Index(
		"pins",
		bytes.NewReader(data),
		es.client.Index.WithDocumentID(strconv.Itoa(pin.ID)),
		es.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error indexing pin %d: %v", pin.ID, err)
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("error indexing pin %d: %s", pin.ID, string(body))
	}

	log.Printf("Successfully indexed pin %d", pin.ID)
	return nil
}

func (es *ESClient) SearchPins(ctx context.Context, query string, tags []string) ([]PinDocument, error) {
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"nested": map[string]interface{}{
							"path": "tags",
							"query": map[string]interface{}{
								"bool": map[string]interface{}{
									"should": []map[string]interface{}{
										{
											"match": map[string]interface{}{
												"tags.title_ru": map[string]interface{}{
													"query":     query,
													"fuzziness": "AUTO",
													"boost":     3,
												},
											},
										},
										{
											"match": map[string]interface{}{
												"tags.title_en": map[string]interface{}{
													"query":     query,
													"fuzziness": "AUTO",
													"boost":     2,
												},
											},
										},
										{
											"match_phrase_prefix": map[string]interface{}{
												"tags.title_ru": map[string]interface{}{
													"query": query,
													"boost": 2,
												},
											},
										},
										{
											"match_phrase_prefix": map[string]interface{}{
												"tags.title_en": map[string]interface{}{
													"query": query,
													"boost": 1,
												},
											},
										},
									},
								},
							},
							"score_mode": "max",
						},
					},
					{
						"match": map[string]interface{}{
							"original_file_name": map[string]interface{}{
								"query":         query,
								"fuzziness":     "AUTO",
								"prefix_length": 2,
								"boost":         2,
							},
						},
					},
					{
						"match": map[string]interface{}{
							"title": map[string]interface{}{
								"query":     query,
								"fuzziness": "AUTO",
								"boost":     3,
							},
						},
					},
					{
						"match": map[string]interface{}{
							"description": map[string]interface{}{
								"query":     query,
								"fuzziness": "AUTO",
								"boost":     1,
							},
						},
					},
					{
						"match_phrase": map[string]interface{}{
							"description": map[string]interface{}{
								"query": query,
								"boost": 2,
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"original_file_name": map[string]interface{}{},
				"title":              map[string]interface{}{},
				"description":        map[string]interface{}{},
				"tags.title_ru":      map[string]interface{}{},
				"tags.title_en":      map[string]interface{}{},
			},
		},
	}

	if len(tags) > 0 {
		tagQueries := make([]map[string]interface{}, len(tags))
		for i, tag := range tags {
			tagQueries[i] = map[string]interface{}{
				"nested": map[string]interface{}{
					"path": "tags",
					"query": map[string]interface{}{
						"bool": map[string]interface{}{
							"should": []map[string]interface{}{
								{
									"match": map[string]interface{}{
										"tags.title_ru": tag,
									},
								},
								{
									"match": map[string]interface{}{
										"tags.title_en": tag,
									},
								},
							},
						},
					},
				},
			}
		}
		searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = tagQueries
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, fmt.Errorf("error encoding query: %v", err)
	}

	log.Printf("Search query: %s", buf.String())

	res, err := es.client.Search(
		es.client.Search.WithContext(ctx),
		es.client.Search.WithIndex("pins"),
		es.client.Search.WithBody(&buf),
		es.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching: %v", err)
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search error: %s", string(body))
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source    PinDocument         `json:"_source"`
				Score     float64             `json:"_score"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	log.Printf("Found %d documents", result.Hits.Total.Value)

	pins := make([]PinDocument, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		pins[i] = hit.Source
		log.Printf("Document %d: score=%f", hit.Source.ID, hit.Score)
		if len(hit.Highlight) > 0 {
			log.Printf("Highlights for document %d: %v", hit.Source.ID, hit.Highlight)
		}
	}

	return pins, nil
}

func (es *ESClient) SearchPinIDs(ctx context.Context, query string) ([]int, int, error) {
	searchQuery := map[string]interface{}{
		"_source": []string{"id"},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"nested": map[string]interface{}{
							"path": "tags",
							"query": map[string]interface{}{
								"bool": map[string]interface{}{
									"should": []map[string]interface{}{
										{
											"match": map[string]interface{}{
												"tags.title_ru": map[string]interface{}{
													"query":     query,
													"fuzziness": "AUTO",
													"boost":     3,
												},
											},
										},
										{
											"match": map[string]interface{}{
												"tags.title_en": map[string]interface{}{
													"query":     query,
													"fuzziness": "AUTO",
													"boost":     2,
												},
											},
										},
									},
								},
							},
							"score_mode": "max",
						},
					},
					{
						"match": map[string]interface{}{
							"title": map[string]interface{}{
								"query":     query,
								"fuzziness": "AUTO",
								"boost":     3,
							},
						},
					},
					{
						"match": map[string]interface{}{
							"description": map[string]interface{}{
								"query":     query,
								"fuzziness": "AUTO",
								"boost":     1,
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, 0, fmt.Errorf("error encoding query: %v", err)
	}

	res, err := es.client.Search(
		es.client.Search.WithContext(ctx),
		es.client.Search.WithIndex("pins"),
		es.client.Search.WithBody(&buf),
		es.client.Search.WithTrackTotalHits(true),
		es.client.Search.WithSize(10000), // Увеличиваем размер выборки для получения всех ID
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("search error: %s", string(body))
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source struct {
					ID int `json:"id"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("error parsing response: %v", err)
	}

	pinIDs := make([]int, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		pinIDs[i] = hit.Source.ID
	}

	return pinIDs, result.Hits.Total.Value, nil
}

package elasticsearch

const PinMapping = `{
        "settings": {
            "analysis": {
                "filter": {
                    "russian_stop": {
                        "type": "stop",
                        "stopwords": "_russian_"
                    },
                    "russian_stemmer": {
                        "type": "stemmer",
                        "language": "russian"
                    },
                    "word_delimiter_custom": {
                        "type": "word_delimiter_graph",
                        "split_on_numerics": true,
                        "split_on_case_change": true,
                        "generate_word_parts": true,
                        "generate_number_parts": true,
                        "preserve_original": true,
                        "catenate_all": true
                    }
                },
                "analyzer": {
                    "russian_custom": {
                        "tokenizer": "standard",
                        "filter": [
                            "lowercase",
                            "russian_stop",
                            "russian_stemmer",
                            "word_delimiter_custom"
                        ]
                    },
                    "filename_analyzer": {
                        "tokenizer": "standard",
                        "filter": [
                            "lowercase",
                            "word_delimiter_custom"
                        ]
                    }
                }
            }
        },
        "mappings": {
            "properties": {
                "id": { "type": "integer" },
                "title": { 
                    "type": "text",
                    "analyzer": "russian_custom",
                    "fields": {
                        "keyword": { "type": "keyword" }
                    }
                },
                "description": { 
                    "type": "text",
                    "analyzer": "russian_custom"
                },
                "original_file_name": { 
                    "type": "text",
                    "analyzer": "filename_analyzer",
                    "fields": {
                        "keyword": { "type": "keyword" }
                    }
                },
                "path": { "type": "keyword" },
                "type": { "type": "keyword" },
                "tags": {
                    "type": "nested",
                    "properties": {
                        "title_en": {
                            "type": "text",
                            "analyzer": "english",
                            "fields": {
                                "keyword": { "type": "keyword" }
                            }
                        },
                        "title_ru": {
                            "type": "text",
                            "analyzer": "russian_custom",
                            "fields": {
                                "keyword": { "type": "keyword" }
                            }
                        }
                    }
                },
                "created_at": { "type": "date" },
                "updated_at": { "type": "date" }
            }
        }
    }`

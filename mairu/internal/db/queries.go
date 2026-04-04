package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/meilisearch/meilisearch-go"
)

type SymbolLocation struct {
	FilePath string `json:"file_path"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	StartRow uint32 `json:"start_row"`
	EndRow   uint32 `json:"end_row"`
}

func (db *DB) FindSymbol(name string) ([]SymbolLocation, error) {
	if db == nil || db.client == nil {
		return nil, errors.New("database client is not initialized")
	}

	resp, err := db.client.Index(symbolsIndexName).Search(name, &meilisearch.SearchRequest{
		Limit: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}

	var locations []SymbolLocation
	for _, h := range resp.Hits {
		bytes, err := json.Marshal(h)
		if err != nil {
			continue
		}

		var loc SymbolLocation
		if err := json.Unmarshal(bytes, &loc); err == nil {
			locations = append(locations, loc)
		}
	}

	return locations, nil
}

package contextsrv

import (
	"encoding/json"
	"github.com/meilisearch/meilisearch-go"
)

type SymbolLocation struct {
	FilePath string `json:"file_path"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	StartRow uint32 `json:"start_row"`
	EndRow   uint32 `json:"end_row"`
}

func (m *MeiliIndexer) FindSymbol(name string) ([]SymbolLocation, error) {
	resp, err := m.client.Index(IndexSymbols).Search(name, &meilisearch.SearchRequest{
		Limit: 10,
	})
	if err != nil {
		return nil, err
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

func (m *MeiliIndexer) InsertSymbol(id, fileID, name, kind string, exported bool, startRow, startCol, endRow, endCol uint32) error {
	document := map[string]interface{}{
		"id":        id,
		"file_path": fileID,
		"name":      name,
		"kind":      kind,
		"exported":  exported,
		"start_row": startRow,
		"start_col": startCol,
		"end_row":   endRow,
		"end_col":   endCol,
	}

	_, err := m.client.Index(IndexSymbols).AddDocuments([]map[string]interface{}{document}, nil)
	return err
}

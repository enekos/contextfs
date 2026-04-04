package contextsrv

import (
	"encoding/json"
	"fmt"

	"github.com/meilisearch/meilisearch-go"
)

const (
	meiliMemoriesIndex = "contextfs_memories"
	meiliSkillsIndex   = "contextfs_skills"
	meiliNodesIndex    = "contextfs_context_nodes"
)

type MeiliIndexer struct {
	client *meilisearch.Client
}

func NewMeiliIndexer(host, apiKey string) *MeiliIndexer {
	return &MeiliIndexer{
		client: meilisearch.NewClient(meilisearch.ClientConfig{
			Host:   host,
			APIKey: apiKey,
		}),
	}
}

func (m *MeiliIndexer) EnsureIndexes() error {
	indexes := []string{meiliMemoriesIndex, meiliSkillsIndex, meiliNodesIndex}
	for _, idx := range indexes {
		_, _ = m.client.CreateIndex(&meilisearch.IndexConfig{Uid: idx})
	}
	return nil
}

func (m *MeiliIndexer) Upsert(entityType string, payload map[string]any) error {
	idx, err := indexFromEntity(entityType)
	if err != nil {
		return err
	}
	_, err = m.client.Index(idx).AddDocuments([]map[string]any{payload})
	return err
}

func (m *MeiliIndexer) Delete(entityType, id string) error {
	idx, err := indexFromEntity(entityType)
	if err != nil {
		return err
	}
	_, err = m.client.Index(idx).DeleteDocument(id)
	return err
}

func decodePayload(raw []byte) (map[string]any, error) {
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func indexFromEntity(entityType string) (string, error) {
	switch entityType {
	case "memory":
		return meiliMemoriesIndex, nil
	case "skill":
		return meiliSkillsIndex, nil
	case "context_node":
		return meiliNodesIndex, nil
	default:
		return "", fmt.Errorf("unsupported entity type %q", entityType)
	}
}

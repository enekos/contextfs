package contextsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (r *SQLiteRepository) CreateContextNode(ctx context.Context, input ContextCreateInput) (ContextNode, error) {
	now := time.Now().UTC()
	if input.CreatedAt != nil {
		now = *input.CreatedAt
	}
	reasonsJSON, err := json.Marshal(input.ModerationReasons)
	if err != nil {
		return ContextNode{}, fmt.Errorf("marshal moderation reasons: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ContextNode{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO context_nodes (uri, project, parent_uri, name, abstract, overview, content, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)
		ON CONFLICT (uri) DO UPDATE SET
			project = excluded.project,
			parent_uri = excluded.parent_uri,
			name = excluded.name,
			abstract = excluded.abstract,
			overview = excluded.overview,
			content = excluded.content,
			metadata = excluded.metadata,
			moderation_status = excluded.moderation_status,
			moderation_reasons = excluded.moderation_reasons,
			review_required = excluded.review_required,
			version = context_nodes.version + 1,
			updated_at = excluded.updated_at
	`, input.URI, input.Project, input.ParentURI, input.Name, input.Abstract, input.Overview, input.Content, jsonString(input.Metadata, `{}`), input.ModerationStatus, string(reasonsJSON), input.ReviewRequired, now)
	if err != nil {
		return ContextNode{}, err
	}
	if err := r.insertModerationEventTx(ctx, tx, "context_node", input.URI, input.Project, input.ModerationStatus, input.ModerationReasons, input.ReviewRequired); err != nil {
		return ContextNode{}, err
	}
	if err := r.insertAuditTx(ctx, tx, "context_node", input.URI, "upsert", "contextsrv", map[string]any{"project": input.Project}); err != nil {
		return ContextNode{}, err
	}
	if err := tx.Commit(); err != nil {
		return ContextNode{}, err
	}
	return ContextNode{
		URI:               input.URI,
		Project:           input.Project,
		ParentURI:         input.ParentURI,
		Name:              input.Name,
		Abstract:          input.Abstract,
		Overview:          input.Overview,
		Content:           input.Content,
		ModerationStatus:  input.ModerationStatus,
		ModerationReasons: input.ModerationReasons,
		ReviewRequired:    input.ReviewRequired,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func (r *SQLiteRepository) GetContextNode(ctx context.Context, uri string) (ContextNode, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT uri, project, parent_uri, name, abstract, overview, content, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at
		FROM context_nodes WHERE uri = $1
	`, uri)
	var n ContextNode
	var reasonsRaw []byte
	var metadataRaw []byte
	if err := row.Scan(&n.URI, &n.Project, &n.ParentURI, &n.Name, &n.Abstract, &n.Overview, &n.Content, &metadataRaw, &n.ModerationStatus, &reasonsRaw, &n.ReviewRequired, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return ContextNode{}, err
	}
	if err := unmarshalJSONField(reasonsRaw, &n.ModerationReasons); err != nil {
		return ContextNode{}, fmt.Errorf("unmarshal moderation_reasons for node %s: %w", n.URI, err)
	}
	if err := unmarshalJSONField(metadataRaw, &n.Metadata); err != nil {
		return ContextNode{}, fmt.Errorf("unmarshal metadata for node %s: %w", n.URI, err)
	}
	return n, nil
}

func (r *SQLiteRepository) ListContextNodes(ctx context.Context, project string, parentURI *string, limit int) ([]ContextNode, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT uri, project, parent_uri, name, abstract, overview, content, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at FROM context_nodes WHERE 1=1`
	args := []any{}
	argN := 1
	if project != "" {
		q += fmt.Sprintf(" AND project = $%d", argN)
		args = append(args, project)
		argN++
	}
	if parentURI != nil {
		q += fmt.Sprintf(" AND parent_uri = $%d", argN)
		args = append(args, *parentURI)
	}
	q += ` ORDER BY created_at DESC LIMIT `
	q += fmt.Sprintf("%d", limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContextNode
	for rows.Next() {
		var n ContextNode
		var reasonsRaw []byte
		var metadataRaw []byte
		if err := rows.Scan(&n.URI, &n.Project, &n.ParentURI, &n.Name, &n.Abstract, &n.Overview, &n.Content, &metadataRaw, &n.ModerationStatus, &reasonsRaw, &n.ReviewRequired, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if err := unmarshalJSONField(reasonsRaw, &n.ModerationReasons); err != nil {
			return nil, fmt.Errorf("unmarshal moderation_reasons for node %s: %w", n.URI, err)
		}
		if err := unmarshalJSONField(metadataRaw, &n.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata for node %s: %w", n.URI, err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) UpdateContextNode(ctx context.Context, input ContextUpdateInput) (ContextNode, error) {
	if input.URI == "" {
		return ContextNode{}, fmt.Errorf("uri is required")
	}

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE context_nodes
		SET name = COALESCE(NULLIF(?, ''), name),
		    abstract = COALESCE(NULLIF(?, ''), abstract),
		    overview = COALESCE(?, overview),
		    content = COALESCE(?, content),
		    updated_at = ?,
		    version = version + 1
		WHERE uri = ?
	`, input.Name, input.Abstract, input.Overview, input.Content, now, input.URI)
	if err != nil {
		return ContextNode{}, err
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT uri, project, parent_uri, name, abstract, overview, content, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at
		FROM context_nodes WHERE uri = $1
	`, input.URI)
	var n ContextNode
	var reasonsRaw []byte
	var metadataRaw []byte
	if err := row.Scan(&n.URI, &n.Project, &n.ParentURI, &n.Name, &n.Abstract, &n.Overview, &n.Content, &metadataRaw, &n.ModerationStatus, &reasonsRaw, &n.ReviewRequired, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return ContextNode{}, err
	}
	if err := unmarshalJSONField(reasonsRaw, &n.ModerationReasons); err != nil {
		return ContextNode{}, fmt.Errorf("unmarshal moderation_reasons for node %s: %w", n.URI, err)
	}
	if err := unmarshalJSONField(metadataRaw, &n.Metadata); err != nil {
		return ContextNode{}, fmt.Errorf("unmarshal metadata for node %s: %w", n.URI, err)
	}
	return n, nil
}

func (r *SQLiteRepository) DeleteContextNode(ctx context.Context, uri string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM context_nodes WHERE uri = $1`, uri)
	return err
}

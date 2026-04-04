package contextsrv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dsn string) (*PostgresRepository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(12)
	db.SetConnMaxLifetime(30 * time.Minute)

	repo := &PostgresRepository{db: db}
	if err := repo.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			project TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			category TEXT NOT NULL,
			owner TEXT NOT NULL,
			importance INT NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			moderation_status TEXT NOT NULL,
			moderation_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
			review_required BOOLEAN NOT NULL DEFAULT false,
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS skills (
			id TEXT PRIMARY KEY,
			project TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			moderation_status TEXT NOT NULL,
			moderation_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
			review_required BOOLEAN NOT NULL DEFAULT false,
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS context_nodes (
			uri TEXT PRIMARY KEY,
			project TEXT NOT NULL DEFAULT '',
			parent_uri TEXT NULL,
			name TEXT NOT NULL,
			abstract TEXT NOT NULL,
			overview TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			moderation_status TEXT NOT NULL,
			moderation_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
			review_required BOOLEAN NOT NULL DEFAULT false,
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS moderation_events (
			id BIGSERIAL PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			project TEXT NOT NULL DEFAULT '',
			decision TEXT NOT NULL,
			reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
			review_status TEXT NOT NULL DEFAULT 'pending',
			reviewer_decision TEXT NOT NULL DEFAULT '',
			reviewer TEXT NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT '',
			policy_version TEXT NOT NULL DEFAULT 'v1',
			review_required BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			reviewed_at TIMESTAMPTZ NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id BIGSERIAL PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			action TEXT NOT NULL,
			actor TEXT NOT NULL DEFAULT 'system',
			details JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS search_outbox (
			id BIGSERIAL PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			op_type TEXT NOT NULL,
			payload JSONB NOT NULL,
			payload_hash TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			retry_count INT NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_project_created ON memories(project, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_project_created ON skills(project, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_project_parent ON context_nodes(project, parent_uri)`,
		`CREATE INDEX IF NOT EXISTS idx_moderation_pending ON moderation_events(review_status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_outbox_pending ON search_outbox(status, next_attempt_at, id)`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) CreateMemory(ctx context.Context, input MemoryCreateInput) (Memory, error) {
	id := fmt.Sprintf("mem_%d", time.Now().UnixNano())
	now := time.Now().UTC()
	reasonsJSON, _ := json.Marshal(input.ModerationReasons)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Memory{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO memories (id, project, content, category, owner, importance, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9::jsonb,$10,$11,$11)
	`, id, input.Project, input.Content, input.Category, input.Owner, input.Importance, jsonString(input.Metadata, `{}`), input.ModerationStatus, string(reasonsJSON), input.ReviewRequired, now)
	if err != nil {
		return Memory{}, err
	}
	if err := r.insertModerationEventTx(ctx, tx, "memory", id, input.Project, input.ModerationStatus, input.ModerationReasons, input.ReviewRequired); err != nil {
		return Memory{}, err
	}
	if err := r.insertAuditTx(ctx, tx, "memory", id, "create", "contextsrv", map[string]any{"project": input.Project}); err != nil {
		return Memory{}, err
	}
	if err := tx.Commit(); err != nil {
		return Memory{}, err
	}
	return Memory{
		ID:                id,
		Project:           input.Project,
		Content:           input.Content,
		Category:          input.Category,
		Owner:             input.Owner,
		Importance:        input.Importance,
		ModerationStatus:  input.ModerationStatus,
		ModerationReasons: input.ModerationReasons,
		ReviewRequired:    input.ReviewRequired,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func (r *PostgresRepository) ListMemories(ctx context.Context, project string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT id, project, content, category, owner, importance, moderation_status, moderation_reasons, review_required, created_at, updated_at FROM memories`
	var args []any
	if project != "" {
		q += ` WHERE project = $1`
		args = append(args, project)
	}
	q += ` ORDER BY created_at DESC LIMIT `
	q += fmt.Sprintf("%d", limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var m Memory
		var reasonsRaw []byte
		if err := rows.Scan(&m.ID, &m.Project, &m.Content, &m.Category, &m.Owner, &m.Importance, &m.ModerationStatus, &reasonsRaw, &m.ReviewRequired, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reasonsRaw, &m.ModerationReasons)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateMemory(ctx context.Context, input MemoryUpdateInput) (Memory, error) {
	if input.ID == "" {
		return Memory{}, fmt.Errorf("id is required")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE memories
		SET content = COALESCE(NULLIF($2, ''), content),
		    category = COALESCE(NULLIF($3, ''), category),
		    owner = COALESCE(NULLIF($4, ''), owner),
		    importance = CASE WHEN $5 > 0 THEN $5 ELSE importance END,
		    updated_at = NOW()
		WHERE id = $1
	`, input.ID, input.Content, input.Category, input.Owner, input.Importance)
	if err != nil {
		return Memory{}, err
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project, content, category, owner, importance, moderation_status, moderation_reasons, review_required, created_at, updated_at
		FROM memories WHERE id = $1
	`, input.ID)
	var m Memory
	var reasonsRaw []byte
	if err := row.Scan(&m.ID, &m.Project, &m.Content, &m.Category, &m.Owner, &m.Importance, &m.ModerationStatus, &reasonsRaw, &m.ReviewRequired, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return Memory{}, err
	}
	_ = json.Unmarshal(reasonsRaw, &m.ModerationReasons)
	return m, nil
}

func (r *PostgresRepository) DeleteMemory(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM memories WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) CreateSkill(ctx context.Context, input SkillCreateInput) (Skill, error) {
	id := fmt.Sprintf("skill_%d", time.Now().UnixNano())
	now := time.Now().UTC()
	reasonsJSON, _ := json.Marshal(input.ModerationReasons)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Skill{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO skills (id, project, name, description, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7::jsonb,$8,$9,$9)
	`, id, input.Project, input.Name, input.Description, jsonString(input.Metadata, `{}`), input.ModerationStatus, string(reasonsJSON), input.ReviewRequired, now)
	if err != nil {
		return Skill{}, err
	}
	if err := r.insertModerationEventTx(ctx, tx, "skill", id, input.Project, input.ModerationStatus, input.ModerationReasons, input.ReviewRequired); err != nil {
		return Skill{}, err
	}
	if err := r.insertAuditTx(ctx, tx, "skill", id, "create", "contextsrv", map[string]any{"project": input.Project}); err != nil {
		return Skill{}, err
	}
	if err := tx.Commit(); err != nil {
		return Skill{}, err
	}
	return Skill{
		ID:                id,
		Project:           input.Project,
		Name:              input.Name,
		Description:       input.Description,
		ModerationStatus:  input.ModerationStatus,
		ModerationReasons: input.ModerationReasons,
		ReviewRequired:    input.ReviewRequired,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func (r *PostgresRepository) ListSkills(ctx context.Context, project string, limit int) ([]Skill, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT id, project, name, description, moderation_status, moderation_reasons, review_required, created_at, updated_at FROM skills`
	var args []any
	if project != "" {
		q += ` WHERE project = $1`
		args = append(args, project)
	}
	q += ` ORDER BY created_at DESC LIMIT `
	q += fmt.Sprintf("%d", limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Skill
	for rows.Next() {
		var s Skill
		var reasonsRaw []byte
		if err := rows.Scan(&s.ID, &s.Project, &s.Name, &s.Description, &s.ModerationStatus, &reasonsRaw, &s.ReviewRequired, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reasonsRaw, &s.ModerationReasons)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateSkill(ctx context.Context, input SkillUpdateInput) (Skill, error) {
	if input.ID == "" {
		return Skill{}, fmt.Errorf("id is required")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE skills
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = COALESCE(NULLIF($3, ''), description),
		    updated_at = NOW()
		WHERE id = $1
	`, input.ID, input.Name, input.Description)
	if err != nil {
		return Skill{}, err
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project, name, description, moderation_status, moderation_reasons, review_required, created_at, updated_at
		FROM skills WHERE id = $1
	`, input.ID)
	var s Skill
	var reasonsRaw []byte
	if err := row.Scan(&s.ID, &s.Project, &s.Name, &s.Description, &s.ModerationStatus, &reasonsRaw, &s.ReviewRequired, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return Skill{}, err
	}
	_ = json.Unmarshal(reasonsRaw, &s.ModerationReasons)
	return s, nil
}

func (r *PostgresRepository) DeleteSkill(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM skills WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) CreateContextNode(ctx context.Context, input ContextCreateInput) (ContextNode, error) {
	now := time.Now().UTC()
	reasonsJSON, _ := json.Marshal(input.ModerationReasons)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ContextNode{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO context_nodes (uri, project, parent_uri, name, abstract, overview, content, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10::jsonb,$11,$12,$12)
		ON CONFLICT (uri) DO UPDATE SET
			project = EXCLUDED.project,
			parent_uri = EXCLUDED.parent_uri,
			name = EXCLUDED.name,
			abstract = EXCLUDED.abstract,
			overview = EXCLUDED.overview,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			moderation_status = EXCLUDED.moderation_status,
			moderation_reasons = EXCLUDED.moderation_reasons,
			review_required = EXCLUDED.review_required,
			version = context_nodes.version + 1,
			updated_at = EXCLUDED.updated_at
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

func (r *PostgresRepository) ListContextNodes(ctx context.Context, project string, parentURI *string, limit int) ([]ContextNode, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT uri, project, parent_uri, name, abstract, overview, content, moderation_status, moderation_reasons, review_required, created_at, updated_at FROM context_nodes WHERE 1=1`
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
		if err := rows.Scan(&n.URI, &n.Project, &n.ParentURI, &n.Name, &n.Abstract, &n.Overview, &n.Content, &n.ModerationStatus, &reasonsRaw, &n.ReviewRequired, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reasonsRaw, &n.ModerationReasons)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateContextNode(ctx context.Context, input ContextUpdateInput) (ContextNode, error) {
	if input.URI == "" {
		return ContextNode{}, fmt.Errorf("uri is required")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE context_nodes
		SET name = COALESCE(NULLIF($2, ''), name),
		    abstract = COALESCE(NULLIF($3, ''), abstract),
		    overview = COALESCE($4, overview),
		    content = COALESCE($5, content),
		    updated_at = NOW(),
		    version = version + 1
		WHERE uri = $1
	`, input.URI, input.Name, input.Abstract, input.Overview, input.Content)
	if err != nil {
		return ContextNode{}, err
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT uri, project, parent_uri, name, abstract, overview, content, moderation_status, moderation_reasons, review_required, created_at, updated_at
		FROM context_nodes WHERE uri = $1
	`, input.URI)
	var n ContextNode
	var reasonsRaw []byte
	if err := row.Scan(&n.URI, &n.Project, &n.ParentURI, &n.Name, &n.Abstract, &n.Overview, &n.Content, &n.ModerationStatus, &reasonsRaw, &n.ReviewRequired, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return ContextNode{}, err
	}
	_ = json.Unmarshal(reasonsRaw, &n.ModerationReasons)
	return n, nil
}

func (r *PostgresRepository) DeleteContextNode(ctx context.Context, uri string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM context_nodes WHERE uri = $1`, uri)
	return err
}

func (r *PostgresRepository) SearchText(ctx context.Context, opts SearchOptions) (map[string]any, error) {
	query := opts.Query
	project := opts.Project
	store := opts.Store
	topK := opts.TopK
	if topK <= 0 {
		topK = 10
	}
	if store == "" {
		store = "all"
	}
	q := "%" + strings.ToLower(query) + "%"
	out := map[string]any{}
	if store == "all" || store == "memories" {
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, content FROM memories
			WHERE ($1 = '' OR project = $1) AND LOWER(content) LIKE $2
			ORDER BY created_at DESC LIMIT $3
		`, project, q, topK)
		if err != nil {
			return nil, err
		}
		var items []map[string]any
		for rows.Next() {
			var id, content string
			if err := rows.Scan(&id, &content); err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, map[string]any{"id": id, "content": content, "_hybrid_score": 0.7})
		}
		rows.Close()
		out["memories"] = items
	}
	if store == "all" || store == "skills" {
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, name, description FROM skills
			WHERE ($1 = '' OR project = $1) AND (LOWER(name) LIKE $2 OR LOWER(description) LIKE $2)
			ORDER BY created_at DESC LIMIT $3
		`, project, q, topK)
		if err != nil {
			return nil, err
		}
		var items []map[string]any
		for rows.Next() {
			var id, name, description string
			if err := rows.Scan(&id, &name, &description); err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, map[string]any{"id": id, "name": name, "description": description, "_hybrid_score": 0.7})
		}
		rows.Close()
		out["skills"] = items
	}
	if store == "all" || store == "context" {
		rows, err := r.db.QueryContext(ctx, `
			SELECT uri, name, abstract FROM context_nodes
			WHERE ($1 = '' OR project = $1) AND (LOWER(name) LIKE $2 OR LOWER(abstract) LIKE $2 OR LOWER(content) LIKE $2)
			ORDER BY created_at DESC LIMIT $3
		`, project, q, topK)
		if err != nil {
			return nil, err
		}
		var items []map[string]any
		for rows.Next() {
			var uri, name, abstract string
			if err := rows.Scan(&uri, &name, &abstract); err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, map[string]any{"uri": uri, "name": name, "abstract": abstract, "_hybrid_score": 0.7})
		}
		rows.Close()
		out["contextNodes"] = items
	}
	return out, nil
}

func (r *PostgresRepository) ListModerationQueue(ctx context.Context, limit int) ([]ModerationEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, entity_type, entity_id, project, decision, reasons, review_status, reviewer_decision, review_required, policy_version, created_at, COALESCE(reviewed_at, '0001-01-01'::timestamptz), reviewer
		FROM moderation_events
		WHERE review_status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModerationEvent
	for rows.Next() {
		var ev ModerationEvent
		var reasonsRaw []byte
		if err := rows.Scan(&ev.ID, &ev.EntityType, &ev.EntityID, &ev.Project, &ev.Decision, &reasonsRaw, &ev.ReviewStatus, &ev.ReviewerDecision, &ev.ReviewRequired, &ev.PolicyVersion, &ev.CreatedAt, &ev.ReviewedAt, &ev.Reviewer); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reasonsRaw, &ev.Reasons)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ReviewModeration(ctx context.Context, input ModerationReviewInput) error {
	if input.EventID == 0 {
		return fmt.Errorf("event_id is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		UPDATE moderation_events
		SET review_status = 'reviewed',
			reviewer_decision = $2,
			reviewer = $3,
			notes = $4,
			reviewed_at = NOW()
		WHERE id = $1
	`, input.EventID, input.Decision, input.Reviewer, input.Notes)
	if err != nil {
		return err
	}
	if err := r.insertAuditTx(ctx, tx, "moderation_event", fmt.Sprintf("%d", input.EventID), "review", input.Reviewer, map[string]any{"decision": input.Decision}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresRepository) EnqueueOutbox(ctx context.Context, entityType, entityID, opType string, payload any) error {
	payloadBytes, _ := json.Marshal(payload)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO search_outbox (entity_type, entity_id, op_type, payload, payload_hash, status, retry_count, next_attempt_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, md5($4::text), 'pending', 0, NOW(), NOW())
	`, entityType, entityID, opType, string(payloadBytes))
	return err
}

func (r *PostgresRepository) insertModerationEventTx(ctx context.Context, tx *sql.Tx, entityType, entityID, project, decision string, reasons []string, reviewRequired bool) error {
	reasonsJSON, _ := json.Marshal(reasons)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO moderation_events (entity_type, entity_id, project, decision, reasons, review_status, review_required, policy_version)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, 'v1')
	`, entityType, entityID, project, decision, string(reasonsJSON), reviewState(reviewRequired), reviewRequired)
	return err
}

func (r *PostgresRepository) insertAuditTx(ctx context.Context, tx *sql.Tx, entityType, entityID, action, actor string, details map[string]any) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log (entity_type, entity_id, action, actor, details)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, entityType, entityID, action, actor, string(detailsJSON))
	return err
}

func reviewState(reviewRequired bool) string {
	if reviewRequired {
		return "pending"
	}
	return "auto_approved"
}

func jsonString(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	return string(raw)
}

package contextsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (r *SQLiteRepository) CreateSkill(ctx context.Context, input SkillCreateInput) (Skill, error) {
	id := fmt.Sprintf("skill_%d", time.Now().UnixNano())
	now := time.Now().UTC()
	reasonsJSON, err := json.Marshal(input.ModerationReasons)
	if err != nil {
		return Skill{}, fmt.Errorf("marshal moderation reasons: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Skill{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO skills (id, project, name, description, metadata, moderation_status, moderation_reasons, review_required, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
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

func (r *SQLiteRepository) ListSkills(ctx context.Context, project string, limit int) ([]Skill, error) {
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
		if err := unmarshalJSONField(reasonsRaw, &s.ModerationReasons); err != nil {
			return nil, fmt.Errorf("unmarshal moderation_reasons for skill %s: %w", s.ID, err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) UpdateSkill(ctx context.Context, input SkillUpdateInput) (Skill, error) {
	if input.ID == "" {
		return Skill{}, fmt.Errorf("id is required")
	}

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE skills
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = COALESCE(NULLIF($3, ''), description),
		    updated_at = $4
		WHERE id = $1
	`, input.ID, input.Name, input.Description, now)
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
	if err := unmarshalJSONField(reasonsRaw, &s.ModerationReasons); err != nil {
		return Skill{}, fmt.Errorf("unmarshal moderation_reasons for skill %s: %w", s.ID, err)
	}
	return s, nil
}

func (r *SQLiteRepository) DeleteSkill(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM skills WHERE id = $1`, id)
	return err
}

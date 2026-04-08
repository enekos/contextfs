package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func FindByID(ctx context.Context, db *sql.DB, id string) (*Record, error) {
	row := db.QueryRowContext(ctx, "SELECT id, name, created_at FROM records WHERE id = ?", id)
	var r Record
	if err := row.Scan(&r.ID, &r.Name, &r.CreatedAt); err != nil {
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return &r, nil
}

func ListRecent(ctx context.Context, db *sql.DB, limit int) ([]Record, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, created_at FROM records ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.ID, &r.Name, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func Create(ctx context.Context, db *sql.DB, name string) (*Record, error) {
	id := generateID()
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, "INSERT INTO records (id, name, created_at) VALUES (?, ?, ?)", id, name, now)
	if err != nil {
		return nil, fmt.Errorf("Create: %w", err)
	}
	return FindByID(ctx, db, id)
}

func Delete(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM records WHERE id = ?", id)
	return err
}

func generateID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}

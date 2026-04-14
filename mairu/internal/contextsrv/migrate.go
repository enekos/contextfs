package contextsrv

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate runs all pending migrations from the embedded migrations directory.
func (r *SQLiteRepository) Migrate(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	if err := r.backfillExistingMigrations(ctx); err != nil {
		return fmt.Errorf("backfill existing migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, err := migrationVersion(entry.Name())
		if err != nil {
			return fmt.Errorf("invalid migration filename %q: %w", entry.Name(), err)
		}

		var applied int
		if err := r.db.QueryRowContext(ctx, `SELECT version FROM schema_migrations WHERE version = ?`, version).Scan(&applied); err == nil {
			// Already applied.
			continue
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %d: %w", version, err)
		}

		content, err := migrationFS.ReadFile(path.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %d: %w", version, err)
		}

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return nil
}

// backfillExistingMigrations detects databases that were created before the
// migration system existed and records their migrations as already applied.
func (r *SQLiteRepository) backfillExistingMigrations(ctx context.Context) error {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return fmt.Errorf("count schema_migrations: %w", err)
	}
	if count > 0 {
		return nil
	}

	// If schema_migrations is empty but the memories table exists and already
	// has the retrieval_count column, the old inline migration logic has run.
	var hasCol int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'retrieval_count'
	`).Scan(&hasCol)
	if err != nil {
		return fmt.Errorf("check memories schema: %w", err)
	}
	if hasCol == 0 {
		// Fresh database — let normal migrations run.
		return nil
	}

	// Existing database — mark all current migrations as applied.
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO schema_migrations (version) VALUES (1), (2)
	`); err != nil {
		return fmt.Errorf("insert backfilled migrations: %w", err)
	}
	return nil
}

func migrationVersion(name string) (int, error) {
	base := strings.TrimSuffix(name, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("expected format NNNN_description.sql")
	}
	return strconv.Atoi(parts[0])
}

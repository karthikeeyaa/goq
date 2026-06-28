package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"goq/migrations"
)

func RunMigrations(ctx context.Context, db *sql.DB, logger *log.Logger) (int, error) {

	// 1. Create schema_migrations metadata table if not exists
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	// 2. Read embedded migrations files
	files, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return 0, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	migrationCount := 0

	// 3. Apply each migration sequentially
	for _, filename := range sqlFiles {
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)"
		err := db.QueryRowContext(ctx, query, filename).Scan(&exists)
		if err != nil {
			return 0, fmt.Errorf("failed to check migration history for %s: %w", filename, err)
		}

		if exists {
			continue
		}

		logger.Printf("Applying database migration: %s", filename)

		content, err := fs.ReadFile(migrations.FS, filename)
		if err != nil {
			return 0, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to execute migration script %s: %w", filename, err)
		}

		recordQuery := "INSERT INTO schema_migrations (version) VALUES (?)"
		if _, err := tx.ExecContext(ctx, recordQuery, filename); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to record migration application for %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return 0, fmt.Errorf("failed to commit migration transaction for %s: %w", filename, err)
		}

		migrationCount += 1
	}

	return migrationCount, nil
}

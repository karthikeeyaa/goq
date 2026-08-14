package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"goq/db/store"
	"goq/internal/logger"
	schema "goq/migrations"
)

func RunMigrations(ctx context.Context, db *sql.DB, log *logger.Logger) error {

	// 1. Read migrations files and create hash key
	files, err := fs.ReadDir(schema.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrations := make(map[string]string)
	var sqlFiles []string

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			content, err := fs.ReadFile(schema.FS, f.Name())
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %w", f.Name(), err)
			}

			hash := sha256.New()
			hash.Write(content)

			migrations[f.Name()] = hex.EncodeToString(hash.Sum(nil))
			sqlFiles = append(sqlFiles, f.Name())
		}
	}

	sort.Strings(sqlFiles)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize a transaction: %w", err)
	}
	defer tx.Rollback()

	queries := store.New(tx)

	// 2. Create migrations table
	err = queries.CreateGoqMigrationTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	migrationCount := 0

	// 3. Apply each migration
	for _, filename := range sqlFiles {

		migrationHash := migrations[filename]
		applied := true

		migration, err := queries.GetGoqMigration(ctx, filename)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				applied = false
			} else {
				return fmt.Errorf("failed to check migration record for %s: %w", filename, err)
			}
		}

		if applied {
			if migrationHash != migration.Hash {
				return fmt.Errorf("migration hash for %s does not match", filename)
			}
			continue
		}

		log.Info("Applying database migration: %s", filename)

		content, err := fs.ReadFile(schema.FS, filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		if err := queries.UpsertGoqMigration(ctx, store.UpsertGoqMigrationParams{
			Name: filename,
			Hash: migrationHash,
		}); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		migrationCount += 1
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	log.Info("Applied %d database migrations.", migrationCount)

	return nil
}

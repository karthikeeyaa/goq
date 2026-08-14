package fixture

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"goq/db/store"
	"goq/internal/config"
	"goq/internal/logger"
)

type Topic struct {
	Name          string `json:"name"`
	RetentionMs   int64  `json:"retention_ms,omitempty"`
	CleanupPolicy string `json:"cleanup_policy,omitempty"`
}

type Fixture struct {
	Topics []Topic `json:"topics"`
}

func CreateFixtures(ctx context.Context, cfg *config.Config, db *sql.DB, log *logger.Logger) (err error) {
	if _, statErr := os.Stat(cfg.FixtureFile); os.IsNotExist(statErr) {
		return fmt.Errorf("fixture file not found: %s", cfg.FixtureFile)
	}

	var fixture Fixture

	bytes, err := os.ReadFile(cfg.FixtureFile)
	if err != nil {
		return fmt.Errorf("unable to read fixture file: %w", err)
	}

	if err := json.Unmarshal(bytes, &fixture); err != nil {
		return fmt.Errorf("unable to unmarshal fixture file: %w", err)
	}

	for _, t := range fixture.Topics {
		if t.Name == "" {
			return fmt.Errorf("fixture contains a topic with an empty name")
		}
		if t.CleanupPolicy != "" && t.CleanupPolicy != "delete" && t.CleanupPolicy != "compact" {
			return fmt.Errorf("topic %q has invalid cleanup_policy %q (must be 'delete' or 'compact')", t.Name, t.CleanupPolicy)
		}
	}

	log.Info("Running fixtures on %d topics", len(fixture.Topics))

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin fixture transaction: %w", err)
	}
	defer tx.Rollback()

	queries := store.New(tx)

	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}

		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("failed to commit fixture transaction: %w", commitErr)
		}
	}()

	for _, t := range fixture.Topics {
		var retentionMs sql.NullInt64
		if t.RetentionMs > 0 {
			retentionMs = sql.NullInt64{Int64: t.RetentionMs, Valid: true}
		}

		var cleanupPolicy sql.NullString
		if t.CleanupPolicy != "" {
			cleanupPolicy = sql.NullString{String: t.CleanupPolicy, Valid: true}
		}

		err = queries.UpsertTopic(ctx, store.UpsertTopicParams{
			Name:          t.Name,
			RetentionMs:   retentionMs,
			CleanupPolicy: cleanupPolicy,
		})

		if err != nil {
			return fmt.Errorf("failed to upsert topic %s: %w", t.Name, err)
		}
	}

	log.Info("Fixture loaded successfully.")

	return nil
}

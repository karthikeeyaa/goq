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
	Name                  string `json:"name"`
	RetentionMs           int64  `json:"retention_ms,omitempty"`
	CleanupPolicy         string `json:"cleanup_policy,omitempty"`
	MaxMessageBytes       int64  `json:"max_message_bytes,omitempty"`
	LogIndexIntervalBytes int64  `json:"log_index_interval_bytes,omitempty"`
}

type Fixture struct {
	Topics []Topic `json:"topics"`
}

func CreateFixtures(ctx context.Context, cfg *config.Config, db *sql.DB, log *logger.Logger) error {
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
			return fmt.Errorf("fixture structure constraint failed: found topic with an empty name identifier")
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

		var maxMsgBytes sql.NullInt64
		if t.MaxMessageBytes > 0 {
			maxMsgBytes = sql.NullInt64{Int64: t.MaxMessageBytes, Valid: true}
		}

		var logIndexInterval sql.NullInt64
		if t.LogIndexIntervalBytes > 0 {
			logIndexInterval = sql.NullInt64{Int64: t.LogIndexIntervalBytes, Valid: true}
		}

		err = queries.UpsertTopic(ctx, store.UpsertTopicParams{
			Name:                  t.Name,
			RetentionMs:           retentionMs,
			CleanupPolicy:         cleanupPolicy,
			MaxMessageBytes:       maxMsgBytes,
			LogIndexIntervalBytes: logIndexInterval,
		})

		if err != nil {
			return fmt.Errorf("failed to upsert topic %s: %w", t.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Fixture loaded successfully.")

	return nil
}

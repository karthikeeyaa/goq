package fixture

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"goq/db/generated"
)

type Topic struct {
	Name             string          `json:"name"`
	Mode             string          `json:"mode"`
	RetentionSeconds int64           `json:"retention"`
	SchemaValidation bool            `json:"schema_validation"`
	SchemaJSON       json.RawMessage `json:"schema_json,omitempty"`
}

type Fixture struct {
	Topics []Topic `json:"topics"`
}

func CreateFixtures(fixtureFile string, db *sql.DB, logger *log.Logger) (err error) {
	if _, statErr := os.Stat(fixtureFile); os.IsNotExist(statErr) {
		return fmt.Errorf("fixture file not found: %s", fixtureFile)
	}

	var fixture Fixture

	bytes, err := os.ReadFile(fixtureFile)
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
		if t.Mode != "pull" && t.Mode != "push" {
			return fmt.Errorf("topic %q has invalid mode %q (must be 'pull' or 'push')", t.Name, t.Mode)
		}
		if t.RetentionSeconds <= 0 {
			return fmt.Errorf("topic %q has invalid retention_seconds %d (must be > 0)", t.Name, t.RetentionSeconds)
		}
		if t.SchemaValidation {
			if len(t.SchemaJSON) == 0 {
				return fmt.Errorf("topic %q has schema_validation=true but no schema_json provided", t.Name)
			}
			if !json.Valid(t.SchemaJSON) {
				return fmt.Errorf("topic %q has malformed schema_json", t.Name)
			}
		}
	}

	logger.Printf("Running fixtures on %d topics", len(fixture.Topics))

	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin fixture transaction: %w", err)
	}

	queries := generated.New(tx)

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
		schemaValidation := int64(0)
		if t.SchemaValidation {
			schemaValidation = 1
		}

		var schemaJSON sql.NullString
		if len(t.SchemaJSON) > 0 {
			schemaJSON = sql.NullString{String: string(t.SchemaJSON), Valid: true}
		}

		err = queries.UpsertTopic(ctx, generated.UpsertTopicParams{
			Name:             t.Name,
			Mode:             t.Mode,
			RetentionSeconds: t.RetentionSeconds,
			SchemaValidation: schemaValidation,
			SchemaJson:       schemaJSON,
		})

		if err != nil {
			return fmt.Errorf("failed to upsert topic %s: %w", t.Name, err)
		}
	}

	logger.Println("Fixture loaded successfully.")

	return nil
}
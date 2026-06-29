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
	Name             string `json:"name"`
	Mode             string `json:"mode"`
	ArchiveFile      string `json:"archive_file"`
	RetentionSeconds int64  `json:"retention"`
}

type Fixture struct {
	Topics []Topic `json:"topics"`
}

func CreateFixtures(fixtureFile string, db *sql.DB, logger *log.Logger) (err error) {
	if _, err := os.Stat(fixtureFile); os.IsNotExist(err) {
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

	logger.Printf("Creating %d topics from fixture.", len(fixture.Topics))

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

	if err = queries.DeleteAllTopics(ctx); err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	logger.Println("Cleared existing topics.")

	for _, t := range fixture.Topics {
		_, err = queries.CreateTopic(ctx, generated.CreateTopicParams{
			Name:             t.Name,
			RetentionSeconds: t.RetentionSeconds,
			ArchiveFile:      sql.NullString{String: t.ArchiveFile, Valid: t.ArchiveFile != ""},
			Mode:             t.Mode,
		})

		if err != nil {
			return fmt.Errorf("failed to insert topic %s: %w", t.Name, err)
		}
	}

	logger.Println("Fixture loaded successfully.")

	return nil
}

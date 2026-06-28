package fixture

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"goq/db/generated"

	"github.com/google/uuid"
)

type RetryConfig struct {
	MaxRetries          int `json:"max_retries"`
	BaseIntervalSeconds int `json:"base_interval_secs"`
	MaxIntervalSeconds  int `json:"max_interval_secs"`
}

type Operation struct {
	Name       string          `json:"name"`
	TopicName  string          `json:"topic_name"`
	SchemaJSON json.RawMessage `json:"schema_json"`
}

type Topic struct {
	Name             string `json:"name"`
	MaxRetries       int    `json:"max_retries"`
	BaseIntervalSecs int    `json:"base_interval_secs"`
	MaxIntervalSecs  int    `json:"max_interval_secs"`
}

type Subscription struct {
	TopicName   string `json:"topic_name"`
	ConsumerURL string `json:"consumer_url"`
	SecretKey   string `json:"secret_key"`
	IsActive    bool   `json:"is_active"`
}

type Fixture struct {
	Topics        []Topic        `json:"topics"`
	Operations    []Operation    `json:"operations"`
	Subscriptions []Subscription `json:"subscriptions"`
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

	logger.Printf("Creating %d topics, %d operations, and %d subscriptions from fixture.",
		len(fixture.Topics), len(fixture.Operations), len(fixture.Subscriptions))

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

	topicIDs := make(map[string]uuid.UUID)

	lookupTopicID := func(topicName string) (uuid.UUID, error) {
		if topicID, ok := topicIDs[topicName]; ok {
			return topicID, nil
		}

		topic, err := queries.GetTopicByName(ctx, topicName)
		if err != nil {
			return uuid.Nil, err
		}

		topicIDs[topicName] = topic.ID
		return topic.ID, nil
	}

	// 1. Remove existing fixture rows.
	if err = queries.DeleteAllSubscriptions(ctx); err != nil {
		return fmt.Errorf("failed to delete subscriptions: %w" , err)
	}

	if err = queries.DeleteAllOperations(ctx); err != nil {
		return fmt.Errorf("failed to delete operations: %w" , err)
	}

	if err = queries.DeleteAllTopics(ctx); err != nil {
		return fmt.Errorf("failed to delete topics: %w" , err)
	}

	logger.Printf("Cleared all existing fixture data from topics, operations, subscriptions")

	// 2. Insert Topics
	for _, t := range fixture.Topics {
		topic, err := queries.InsertTopic(ctx, generated.InsertTopicParams{
			ID:               uuid.New(),
			Name:             t.Name,
			MaxRetries:       t.MaxRetries,
			BaseIntervalSecs: t.BaseIntervalSecs,
			MaxIntervalSecs:  t.MaxIntervalSecs,
		})
		if err != nil {
			return fmt.Errorf("failed to insert topic %s: %w", t.Name, err)
		}

		topicIDs[t.Name] = topic.ID
	}

	// 3. Insert Operations
	for _, op := range fixture.Operations {
		topicID, err := lookupTopicID(op.TopicName)
		if err != nil {
			return fmt.Errorf("operation %s references unknown topic %s: %w", op.Name, op.TopicName, err)
		}

		if _, err = queries.InsertOperation(ctx, generated.InsertOperationParams{
			ID:         uuid.New(),
			TopicID:    topicID,
			Name:       op.Name,
			SchemaJson: string(op.SchemaJSON),
		}); err != nil {
			return fmt.Errorf("failed to insert operation %s: %w", op.Name, err)
		}
	}

	// 4. Insert Subscriptions
	for _, sub := range fixture.Subscriptions {
		topicID, err := lookupTopicID(sub.TopicName)
		if err != nil {
			return fmt.Errorf("subscription references unknown topic %s: %w", sub.TopicName, err)
		}

		subscriptionID := uuid.New()

		if _, err = queries.InsertSubscription(ctx, generated.InsertSubscriptionParams{
			ID:          subscriptionID,
			TopicID:     topicID,
			ConsumerUrl: sub.ConsumerURL,
			SecretKey:   sub.SecretKey,
			IsActive:    sub.IsActive,
		}); err != nil {
			return fmt.Errorf("failed to insert subscription for topic %s and consumer %s: %w", sub.TopicName, sub.ConsumerURL, err)
		}
	}

	return nil
}

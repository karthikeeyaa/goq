package models

import (
	"database/sql"
	"time"
)

type RetryConfig struct {
	MaxRetries          int `json:"max_retries"`
	BaseIntervalSeconds int `json:"base_interval_seconds"`
	MaxIntervalSeconds  int `json:"max_interval_seconds"`
}


type Topic struct {
	Name        string      `json:"name"`
	RetryConfig RetryConfig `json:"retry_config"`
	CreatedAt   time.Time   `json:"created_at"`
}


type Operation struct {
	Name       string    `json:"name"`
	TopicName  string    `json:"topic_name"`
	SchemaJSON string    `json:"schema_json"`
	CreatedAt  time.Time `json:"created_at"`
}


type Subscription struct {
	ID          string    `json:"id"`
	TopicName   string    `json:"topic_name"`
	ConsumerURL string    `json:"consumer_url"`
	SecretKey   string    `json:"secret_key"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}


type MessageStatus string

const (
	MessageStatusPending         MessageStatus = "pending"
	MessageStatusProcessing      MessageStatus = "processing"
	MessageStatusSucceeded       MessageStatus = "succeeded"
	MessageStatusPartiallyFailed MessageStatus = "partially_failed"
	MessageStatusFailed          MessageStatus = "failed"
)


type Message struct {
	ID            string        `json:"id"`
	TopicName     string        `json:"topic_name"`
	OperationName string        `json:"operation_name"`
	Payload       string        `json:"payload"`
	Status        MessageStatus `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
}


type AttemptStatus string

const (
	AttemptStatusPending   AttemptStatus = "pending"
	AttemptStatusInFlight  AttemptStatus = "in_flight"
	AttemptStatusSucceeded AttemptStatus = "succeeded"
	AttemptStatusFailed    AttemptStatus = "failed"
	AttemptStatusDead      AttemptStatus = "dead"
)


type DeliveryAttempt struct {
	ID             string         `json:"id"`
	MessageID      string         `json:"message_id"`
	SubscriptionID string         `json:"subscription_id"`
	AttemptNumber  int            `json:"attempt_number"`
	StatusCode     sql.NullInt32  `json:"status_code"`
	ErrorMessage   sql.NullString `json:"error_message"`
	NextRetryAt    sql.NullTime   `json:"next_retry_at"`
	Status         AttemptStatus  `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
}


type DLQ struct {
	ID             string       `json:"id"`
	MessageID      string       `json:"message_id"`
	SubscriptionID string       `json:"subscription_id"`
	LastAttemptID  string       `json:"last_attempt_id"`
	Reason         string       `json:"reason"`
	ReplayedAt     sql.NullTime `json:"replayed_at"`
	CreatedAt      time.Time    `json:"created_at"`
}

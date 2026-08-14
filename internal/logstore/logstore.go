package logstore

import (
	"context"
	"database/sql"
	"path/filepath"
	"os"
	"sync"

	"goq/db/store"
	"goq/internal/config"
	"goq/internal/logger"
)

type MessageFile struct {
	mu    sync.Mutex
	Topic store.Topic
	Path  string
	File  *os.File
}

type MessageStore map[string]*MessageFile

func Init(ctx context.Context, cfg *config.Config, database *sql.DB, log *logger.Logger) (*MessageStore, error) {

	queries := store.New(database)

	topics, err := queries.ListTopics(ctx)
	if err != nil {
		return nil, err
	}

	messageStore := make(MessageStore)

	if err := os.MkdirAll(cfg.MessagesDir, 0755); err != nil {
		return nil, err
	}

	for _, topic := range topics {

		filePath := filepath.Join(cfg.MessagesDir, topic.Name+".log")

		file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}

		messageStore[topic.Name] = &MessageFile{
			Topic: topic,
			Path:  filePath,
			File:  file,
		}
	}
	

	return &messageStore, nil
}

func (ms *MessageStore) Close() error {
	for _, mf := range *ms {
		mf.mu.Lock()
		if mf.File != nil {
			mf.File.Close()
		}
		mf.mu.Unlock()
	}
	return nil
}
package logstore

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	"goq/db/store"
	"goq/internal/config"
	"goq/internal/logger"
)

/**
Log file stores the messages in an append only scheme, where a payload sent by producer
is encoded to bytes and appended to the segment log along with a header containing metadata.

The Payload has a max length constraint of config.MAX_MESSAGE_BYTES, combination of header and payload is message in log file
Every message can have a varying length as the payload length is varying.

Segmenting is not done in GOQ to keep the architecture simple

Log data layout (Variable-Length Records)
- CRC32 Checksum        : uint32 (4 bytes)
- Payload length        : uint32 (4 bytes)
- Timestamp MS 					: int64  (8 bytes)
- Payload               : Raw Bytes (N bytes)

Index data layout (Fixed-Length Records - Exactly 12 bytes per record)
- Absolute position     : uint64 (8 bytes) -> Byte offset where the message starts in the .log file
- Payload length        : uint32 (4 bytes) -> Exact width to read from the .log file


Seek Logic:
Every entry in the index file is exactly 12 bytes
	Index File Seek Position = Target Offset * 12 Bytes

From index file seek, get Absolute position and Payload length which allows to
directly seek into message body in the log file
*/

type LogFile struct {
	mu        sync.Mutex
	Topic     store.Topic
	File      *os.File
	IndexFile *os.File
}

type LogStore map[string]*LogFile

func Init(ctx context.Context, cfg *config.Config, database *sql.DB, log *logger.Logger) (*LogStore, error) {

	queries := store.New(database)

	topics, err := queries.ListTopics(ctx)
	if err != nil {
		return nil, err
	}

	logStore := make(LogStore)

	if err := os.MkdirAll(cfg.MessagesDir, 0755); err != nil {
		return nil, err
	}

	for _, topic := range topics {

		filePath := filepath.Join(cfg.MessagesDir, topic.Name+".log")
		indexFilePath := filepath.Join(cfg.MessagesDir, topic.Name+".index")

		file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}

		indexFile, err := os.OpenFile(indexFilePath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			file.Close()
			return nil, err
		}

		logStore[topic.Name] = &LogFile{
			Topic:     topic,
			File:      file,
			IndexFile: indexFile,
		}
	}

	return &logStore, nil
}

func (ls *LogStore) GetLogFile(topicName string) (*LogFile, bool) {
	lf, exists := (*ls)[topicName]
	return lf, exists
}

func (ls *LogStore) Close() error {
	for _, lf := range *ls {
		lf.Close()
	}
	return nil
}

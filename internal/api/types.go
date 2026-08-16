package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type ErrorCode string

const (
	InvalidInput        ErrorCode = "INVALID_INPUT"
	TopicNotFound       ErrorCode = "TOPIC_NOT_FOUND"
	OffsetOutOfBounds   ErrorCode = "OFFSET_OUT_OF_BOUNDS"
	InternalServerError ErrorCode = "INTERNAL_SERVER_ERROR"
	Unauthorized        ErrorCode = "UNAUTHORIZED"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Status        string    `json:"status"`
	Code          ErrorCode `json:"code"`
	Message       string    `json:"message"`
	HighWatermark int64     `json:"high_watermark"`
}

type PushRequest struct {
	Payload json.RawMessage `json:"payload"`
}

type PushResponse struct {
	Status               string `json:"status"`
	Topic                string `json:"topic"`
	Offset               int64  `json:"offset"`
	MaxMessageBytesLimit int64  `json:"max_message_bytes_limit"`
	TimestampMS          int64  `json:"timestamp_ms"`
}

type PullMessage struct {
	Offset      int64           `json:"offset"`
	TimestampMS int64           `json:"timestamp_ms"`
	Payload     json.RawMessage `json:"payload"`
}

type PullResponse struct {
	Status          string        `json:"status"`
	Count           int           `json:"count"`
	Topic           string        `json:"topic"`
	RequestedOffset int64         `json:"requested_offset"`
	NextOffset      int64         `json:"next_offset"`
	HighWatermark   int64         `json:"high_watermark"`
	Results         []PullMessage `json:"results"`
}

func Response(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func parseQueryInt(r *http.Request, key string, defaultValue int) (int, error) {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return -1, err
	}
	return val, nil
}

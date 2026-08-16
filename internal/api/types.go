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
	InternalServerError ErrorCode = "INTERNAL_SERVER_ERROR"
	Unauthorized        ErrorCode = "UNAUTHORIZED"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Status  string    `json:"status"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type PushRequest struct {
	Payload json.RawMessage `json:"payload"`
}

type PushResponse struct {
	Topic  string `json:"topic"`
	Offset int64  `json:"offset"`
}

type PullMessage struct {
	Offset    int64       `json:"offset"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type PullResponse struct {
	Status  string    `json:"status"`
	Count   int       `json:"count"`
	Results []PullMessage `json:"results"`
}

func Response(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func parseQueryInt64(r *http.Request, key string, defaultValue int64) (int64, error) {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return -1, err
	}
	return val, nil
}

package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type ErrorCode string

const (
	ErrCodeInvalidInput        ErrorCode = "INVALID_INPUT"
	ErrCodeTopicNotFound       ErrorCode = "TOPIC_NOT_FOUND"
	ErrCodeInternalServerError ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
)

type SuccessResponse struct {
	Status  string `json:"status"`
	Count   *int   `json:"count,omitempty"`
	Results any    `json:"results,omitempty"`
}

type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type ErrorResponse struct {
	Status string      `json:"status"`
	Error  ErrorDetail `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, status int, results any) {
	writeJSON(w, status, SuccessResponse{
		Status:  "success",
		Results: results,
	})
}

func writeSuccessWithCount(w http.ResponseWriter, status int, results any, count int) {
	writeJSON(w, status, SuccessResponse{
		Status:  "success",
		Count:   &count,
		Results: results,
	})
}

func writeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	writeJSON(w, status, ErrorResponse{
		Status: "error",
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func parseQueryInt64(r *http.Request, key string, defaultValue int64) (int64, bool, error) {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue, false, nil
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return 0, true, err
	}
	return val, true, nil
}

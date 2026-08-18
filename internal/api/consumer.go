package api

import (
	"net/http"

	"goq/db/store"
	"goq/internal/logstore"
)

func Pull(queries *store.Queries, logStore *logstore.LogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		topic := topicFromCtx(r.Context())

		logFile, exists := logStore.GetLogFile(topic.Name)
		if !exists {
			Response(w, http.StatusNotFound, ErrorResponse{
				Status:  "error",
				Code:    LogFileNotFound,
				Message: "log file not found for topic " + topic.Name,
			})
			return
		}

		offset, err := parseQueryInt(r, "offset", 0)
		if err != nil || offset < 0 {
			Response(w, http.StatusBadRequest, ErrorResponse{
				Status:  "error",
				Code:    InvalidInput,
				Message: "invalid 'from' query parameter",
			})
			return
		}

		limit, err := parseQueryInt(r, "limit", 100)
		if err != nil || limit <= 0 {
			Response(w, http.StatusBadRequest, ErrorResponse{
				Status:  "error",
				Code:    InvalidInput,
				Message: "invalid 'limit' query parameter",
			})
			return
		}
		if limit > 1000 {
			limit = 1000
		}

		logMessages, highWatermark, err := logFile.Read(offset, limit)
		if err != nil {
			Response(w, http.StatusInternalServerError, ErrorResponse{
				Status:  "error",
				Code:    InternalServerError,
				Message: "failed to read messages from log file",
			})
			return
		}

		Response(w, http.StatusOK, PullResponse{
			Status:          "success",
			Count:           len(logMessages),
			Topic:           topic.Name,
			RequestedOffset: offset,
			NextOffset:      offset + len(logMessages),
			HighWatermark:   highWatermark,
			Results:         logMessages,
		})
	}
}

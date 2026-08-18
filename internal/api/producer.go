package api

import (
	"encoding/json"
	"net/http"

	"goq/db/store"
	"goq/internal/logstore"
)

func Push(queries *store.Queries, logStore *logstore.LogStore) http.HandlerFunc {
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

		var req PushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Payload) == 0 {
			Response(w, http.StatusBadRequest, ErrorResponse{
				Status:  "error",
				Code:    InvalidInput,
				Message: "invalid or missing 'payload' in request body",
			})
			return
		}

		logMessage, err := logFile.Append(req.Payload)
		if err != nil {
			Response(w, http.StatusInternalServerError, ErrorResponse{
				Status:  "error",
				Code:    InternalServerError,
				Message: "failed to append message to log file",
			})
			return
		}

		Response(w, http.StatusAccepted, PushResponse{
			Status:               "success",
			Topic:                topic.Name,
			Offset:               logMessage.Offset,
			MaxMessageBytesLimit: int(topic.MaxMessageBytes.Int64),
			TimestampMS:          int(logMessage.TimestampMS),
		})
	}
}

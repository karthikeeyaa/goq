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

		_, exists := logStore.GetLogFile(topic.Name)
		if !exists {
			Response(w, http.StatusNotFound, ErrorResponse{
				Status:  "error",
				Code:    TopicNotFound,
				Message: "topic not found",
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

		// TODO: Write message payload to topic storage segment and receive assigned offset
		res := PushResponse{
			Topic:  topic.Name,
			Offset: 0,
		}

		Response(w, http.StatusAccepted, res)
	}
}

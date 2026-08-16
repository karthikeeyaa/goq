package api

import (
	"net/http"

	"goq/db/store"
	"goq/internal/logstore"
)

func Pull(queries *store.Queries, messageStore *logstore.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		topic := topicFromCtx(r.Context())

		_, exists := messageStore.GetMessageFile(topic.Name)
		if !exists {
			Response(w, http.StatusNotFound, ErrorResponse{
				Status:  "error",
				Code:    TopicNotFound,
				Message: "topic not found",
			})
			return
		}

		from, err := parseQueryInt64(r, "from", 0)
		if err != nil || from < 0 {
			Response(w, http.StatusBadRequest, ErrorResponse{
				Status:  "error",
				Code:    InvalidInput,
				Message: "invalid 'from' query parameter",
			})
			return
		}

		limit, err := parseQueryInt64(r, "limit", 100)
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

		_ = topic
		// TODO: Fetch messages from topic log starting from offset 'from' up to 'limit'
		res := PullResponse{
			Status: "success",
			Count:   0,
			Results: []PullMessage{},
		}

		Response(w, http.StatusOK, res)
	}
}

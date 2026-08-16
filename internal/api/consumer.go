package api

import (
	"net/http"

	"goq/db/store"
	"goq/internal/logstore"
)

func Pull(queries *store.Queries, logStore *logstore.LogStore) http.HandlerFunc {
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

		res := PullResponse{
			Status:  "success",
			Count:   0,
			Results: []PullMessage{},
		}

		Response(w, http.StatusOK, res)
	}
}

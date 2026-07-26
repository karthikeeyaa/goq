package api

import (
	"encoding/json"
	"net/http"

	"goq/db/store"
)

type PullMessageItem struct {
	Offset    int64           `json:"offset"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type PullResponse struct {
	HeadOffset int64             `json:"head_offset"`
	Messages   []PullMessageItem `json:"messages"`
}

func Pull(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topic := topicFromCtx(r.Context())

		from, fromProvided, err := parseQueryInt64(r, "from", 0)
		if err != nil || (fromProvided && from < 0) {
			writeError(w, http.StatusBadRequest, ErrCodeInvalidInput, "invalid 'from' query parameter")
			return
		}

		limit, _, err := parseQueryInt64(r, "limit", 100)
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, ErrCodeInvalidInput, "invalid 'limit' query parameter")
			return
		}
		if limit > 1000 {
			limit = 1000
		}

		_ = topic
		// TODO: Fetch messages from topic log segment starting from offset 'from' up to 'limit'
		messages := []PullMessageItem{}
		res := PullResponse{
			HeadOffset: 0,
			Messages:   messages,
		}

		writeSuccessWithCount(w, http.StatusOK, res, len(res.Messages))
	}
}

package api

import (
	"encoding/json"
	"net/http"

	"goq/db/store"
)

type PullMessageItem struct {
	Offset    int64           `json:"offset"`
	Payload   json.RawMessage `json:"payload"`
}

type PullResponse struct {
	Messages   []PullMessageItem `json:"messages"`
	HeadOffset int64             `json:"head_offset"`
}

func Pull(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Not implemented yet
	}
}

package api

import (
	"encoding/json"
	"net/http"

	"goq/db/store"
)

type PushRequest struct {
	Payload json.RawMessage `json:"payload"`
}

type PushResponse struct {
	Topic  string `json:"topic"`
	Offset int64  `json:"offset"`
}

func Push(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topic := topicFromCtx(r.Context())

		var req PushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Payload) == 0 {
			writeError(w, http.StatusBadRequest, ErrCodeInvalidInput, "invalid or missing 'payload' in request body")
			return
		}

		// TODO: Write message payload to topic storage segment and receive assigned offset
		res := PushResponse{
			Topic:  topic.Name,
			Offset: 0,
		}

		writeSuccess(w, http.StatusAccepted, res)
	}
}

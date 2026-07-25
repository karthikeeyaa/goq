package api

import (
	"net/http"

	"goq/db/store"
)

type PublishResponse struct {
	Topic  string `json:"topic"`
	Offset int64  `json:"offset"`
}

func Publish(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Not implemented yet
	}
}

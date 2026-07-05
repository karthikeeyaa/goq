package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"goq/db/generated"
	"goq/internal/logger"
)

const DEFAULT_PULL_LIMIT int64 = 100

type PullMessageItem struct {
	Offset    int64            `json:"offset"`
	Payload   json.RawMessage  `json:"payload"`
	CreatedAt string `json:"created_at"`
}

type PullResponse struct {
	Topic    string            `json:"topic"`
	Messages []PullMessageItem `json:"messages"`
	Head     int64             `json:"head"`
	Tail     int64             `json:"tail"`
	Count    int               `json:"count"`
}

// Pull handles GET /api/v1/pull/{topic}?from={id}&limit={n}

//   - from=head → lowest offset still available in database
//   - from={id}     → return messages with offset > {id}, ascending, up to limit

func Pull(queries *generated.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topic := topicFromCtx(r.Context())

		fromParam := r.URL.Query().Get("from")
		limitParam := r.URL.Query().Get("limit")

		limit := DEFAULT_PULL_LIMIT
		if limitParam != "" {
			parsed, err := strconv.ParseInt(limitParam, 10, 64)
			if err != nil || parsed <= 0 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "limit must be a positive integer",
				})
				return
			}
			limit = parsed
		}

		// fetch head (earliest/min offset) and tail (latest/max offset)
		headRaw, err := queries.GetEarliestOffset(r.Context(), topic.Name)
		if err != nil {
			logger.Error("failed to get head offset for topic %q: %v", topic.Name, err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "internal server error",
			})
			return
		}
		head := toInt64(headRaw)

		tailRaw, err := queries.GetHeadOffset(r.Context(), topic.Name)
		if err != nil {
			logger.Error("failed to get tail offset for topic %q: %v", topic.Name, err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "internal server error",
			})
			return
		}
		tail := toInt64(tailRaw)

		// ── from omitted: return head/tail with zero messages ──
		if fromParam == "" {
			writeJSON(w, http.StatusOK, PullResponse{
				Topic:    topic.Name,
				Messages: []PullMessageItem{},
				Head:     head,
				Tail:     tail,
				Count:    0,
			})
			return
		}

		// ── from=earliest: use head (already fetched) ──
		var fromOffset int64
		if fromParam == "earliest" {
			// messages with offset >= head
			if head > 0 {
				fromOffset = head - 1
			}
		} else {
			parsed, err := strconv.ParseInt(fromParam, 10, 64)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "from must be a positive integer or 'earliest'",
				})
				return
			}
			fromOffset = parsed
		}

		messages, err := queries.PullMessages(r.Context(), generated.PullMessagesParams{
			TopicName: topic.Name,
			Offset:    fromOffset,
			Limit:     limit,
		})
		if err != nil {
			logger.Error("failed to pull messages from topic %q: %v", topic.Name, err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "internal server error",
			})
			return
		}

		items := make([]PullMessageItem, 0, len(messages))
		for _, m := range messages {
			decoded, _ := base64.StdEncoding.DecodeString(string(m.Payload))
			items = append(items, PullMessageItem{
				Offset:    m.Offset,
				Payload:   json.RawMessage(decoded),
				CreatedAt: m.CreatedAt.Format(time.RFC3339),
			})
		}

		writeJSON(w, http.StatusOK, PullResponse{
			Topic:    topic.Name,
			Messages: items,
			Head:     head,
			Tail:     tail,
			Count:    len(items),
		})
	}
}

// toInt64 converts the interface{} returned by COALESCE queries to int64.
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	default:
		return 0
	}
}
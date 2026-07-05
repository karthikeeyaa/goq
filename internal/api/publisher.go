package api

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/xeipuuv/gojsonschema"

	"goq/db/generated"
	"goq/internal/logger"
)

type PublishResponse struct {
	Offset    int64  `json:"offset"`
	TopicName string `json:"topic_name"`
}

// Publish handles POST /api/v1/publish/{topic}

//  1. Read the entire request body as the raw payload.
//  2. If the topic has schema_validation enabled, validate against schema_json.
//  3. Append the raw payload bytes to the messages table.

func Publish(queries *generated.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topic := topicFromCtx(r.Context())

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "failed to read request body",
			})
			return
		}
		defer r.Body.Close()

		if len(payload) == 0 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "payload is required",
			})
			return
		}

		// schema validation
		if topic.SchemaValidation == 1 && topic.SchemaJson.Valid {
			schemaLoader := gojsonschema.NewStringLoader(topic.SchemaJson.String)
			documentLoader := gojsonschema.NewBytesLoader(payload)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			if err != nil {
				logger.Error("schema validation error for topic %q: %v", topic.Name, err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "schema validation failed unexpectedly",
				})
				return
			}

			if !result.Valid() {
				errors := make([]string, 0, len(result.Errors()))
				for _, e := range result.Errors() {
					errors = append(errors, e.String())
				}
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
					"error":             "payload failed schema validation",
					"validation_errors": errors,
				})
				return
			}
		}

		encoded := base64.StdEncoding.EncodeToString(payload)

		msg, err := queries.CreateMessage(r.Context(), generated.CreateMessageParams{
			TopicName: topic.Name,
			Payload:   []byte(encoded),
		})
		if err != nil {
			logger.Error("failed to persist message to topic %q: %v", topic.Name, err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to persist message",
			})
			return
		}

		writeJSON(w, http.StatusAccepted, PublishResponse{
			Offset:    msg.Offset,
			TopicName: msg.TopicName,
		})
	}
}

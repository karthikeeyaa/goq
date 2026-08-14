package api

import (
	"database/sql"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goq/db/store"
	"goq/internal/logger"
)

// set topic to context
func ValidateTopic(queries *store.Queries, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			topicName := chi.URLParam(r, "topic")
			if topicName == "" {
				writeError(w, http.StatusBadRequest, ErrCodeInvalidInput, "topic name is required")
				return
			}

			topic, err := queries.GetTopic(r.Context(), topicName)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, ErrCodeTopicNotFound, "topic not found: "+topicName)
				return
			}
			if err != nil {
				log.Error("failed to look up topic %q: %v", topicName, err)
				writeError(w, http.StatusInternalServerError, ErrCodeInternalServerError, "internal server error")
				return
			}

			ctx := withTopic(r.Context(), topic)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if reqID != "" {
				log.Info("[%s] %s %s %s", reqID, r.Method, r.URL.Path, ip)
			} else {
				log.Info("%s %s %s", r.Method, r.URL.Path, ip)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware(integrationKey, mode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if mode == "development" || integrationKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			expectedHeader := "token " + integrationKey

			if authHeader != expectedHeader {
				writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "unauthorized")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

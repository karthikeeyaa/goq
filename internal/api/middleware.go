package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"

	"goq/db/store"
	"goq/internal/logger"
)

// set topic to context
func ValidateTopic(queries *store.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			topicName := chi.URLParam(r, "topic")
			if topicName == "" {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "topic name is required",
				})
				return
			}

			topic, err := queries.GetTopic(r.Context(), topicName)
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusNotFound, ErrorResponse{
					Error: "topic not found: " + topicName,
				})
				return
			}
			if err != nil {
				logger.Error("failed to look up topic %q: %v", topicName, err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "internal server error",
				})
				return
			}

			ctx := withTopic(r.Context(), topic)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestLogger(l *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			l.Printf("%s %s %s", r.Method, r.URL.Path, ip)
			next.ServeHTTP(w, r)
		})
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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
				writeJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: "unauthorized",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

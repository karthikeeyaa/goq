package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goq/db/generated"
	"goq/internal/config"
)

func StartServer(cfg *config.Config, database *sql.DB, logger *log.Logger) error {
	queries := generated.New(database)
	router := chi.NewRouter()

	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(RequestLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(time.Duration(cfg.HTTPTimeoutSeconds) * time.Second))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/publish/{topic}", func(r chi.Router) {
			r.Use(TopicValidation(queries))
			r.Post("/", Publish(queries))
		})

		r.Route("/pull/{topic}", func(r chi.Router) {
			r.Use(TopicValidation(queries))
			r.Get("/", Pull(queries))
		})
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Printf("HTTP server listening on %s", addr)
	
	return http.ListenAndServe(addr, router)
}
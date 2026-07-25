package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goq/db/store"
	"goq/internal/config"
)

func StartServer(cfg *config.Config, database *sql.DB, logger *log.Logger) error {
	queries := store.New(database)
	router := chi.NewRouter()

	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(RequestLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(time.Duration(cfg.HTTPTimeoutSeconds) * time.Second))

	router.Get("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	router.Route("/api/"+cfg.Version, func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.IntegrationKey, cfg.Mode))

		r.Route("/publish/{topic}", func(r chi.Router) {
			r.Use(ValidateTopic(queries))
			r.Post("/", Publish(queries))
		})

		r.Route("/pull/{topic}", func(r chi.Router) {
			r.Use(ValidateTopic(queries))
			r.Get("/", Pull(queries))
		})
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				conn.Close()
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Printf("HTTP server listening on %s", addr)

	return http.ListenAndServe(addr, router)
}

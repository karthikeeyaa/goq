package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goq/db/store"
	"goq/internal/config"
)

func StartServer(ctx context.Context, cfg *config.Config, database *sql.DB, logger *log.Logger) error {
	queries := store.New(database)
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(RequestLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(time.Duration(cfg.HTTPTimeoutSeconds) * time.Second))

	router.Get("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{
			"health": "ok",
		})
	})

	router.Route("/api/"+cfg.Version, func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.IntegrationKey, cfg.Mode))

		r.Route("/push/{topic}", func(r chi.Router) {
			r.Use(ValidateTopic(queries))
			r.Post("/", Push(queries))
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

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Printf("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("HTTP server error: %w", err)
	case <-ctx.Done():
		logger.Printf("Shutdown signal received")

		shutdownTimeout := max(time.Duration(cfg.HTTPTimeoutSeconds)*time.Second, 5*time.Second)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			server.Close()
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
		logger.Printf("HTTP server stopped")
	}

	return nil
}

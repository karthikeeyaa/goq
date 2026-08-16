package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goq/db/store"
	"goq/internal/config"
	"goq/internal/logger"
	"goq/internal/logstore"
)

func StartServer(ctx context.Context, cfg *config.Config, database *sql.DB, log *logger.Logger, logStore *logstore.LogStore) error {

	queries := store.New(database)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(RequestLogger(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(time.Duration(cfg.HTTPTimeoutSeconds) * time.Second))

	router.Get("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		Response(w, http.StatusOK, HealthResponse{Status: "ok"})
	})

	router.Route("/api/"+cfg.Version, func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.IntegrationKey, cfg.Mode))

		r.Route("/push/{topic}", func(r chi.Router) {
			r.Use(ValidateTopic(queries, log))
			r.Post("/", Push(queries, logStore))
		})

		r.Route("/pull/{topic}", func(r chi.Router) {
			r.Use(ValidateTopic(queries, log))
			r.Get("/", Pull(queries, logStore))
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
		log.Info("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("HTTP server error: %w", err)
	case <-ctx.Done():
		log.Info("Shutdown signal received")

		shutdownTimeout := max(time.Duration(cfg.HTTPTimeoutSeconds)*time.Second, 5*time.Second)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			server.Close()
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
		log.Info("HTTP server stopped")
	}

	return nil
}

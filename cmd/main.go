package main

import (
	"context"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/sync/semaphore"

	"queuego/internal/config"
	"queuego/internal/db"
	"queuego/internal/logger"
	_ "queuego/migrations"
)

func main() {
	cfg := config.LoadConfig()

	if err := logger.Init(cfg.LogFile); err != nil {
		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Starting %s Message Broker on port %s...", cfg.AppName, cfg.Port)

	database, err := db.Connect(cfg.DBDSN)
	if err != nil {
		logger.Error("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database); err != nil {
		logger.Error("Database migration failed: %v", err)
		os.Exit(1)
	}
	logger.Info("Database migrations applied successfully.")

	_ = chi.NewRouter()
	_ = uuid.New()
	_ = gojsonschema.NewGoLoader(nil)
	_ = semaphore.NewWeighted(1)

	logger.Info("All systems initialized successfully.")
}

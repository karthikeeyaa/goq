package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"goq/internal/api"
	"goq/internal/config"
	"goq/internal/db"
	"goq/internal/fixture"
	"goq/internal/logger"
	_ "goq/schema"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.LoadConfig()

	if err := logger.Init(cfg.LogFile); err != nil {
		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Starting %s Message Broker on port %s...", cfg.AppName, cfg.Port)

	database, err := db.Connect(cfg.DBDSN, logger.InfoLogger)
	if err != nil {
		logger.Error("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	count, err := db.RunMigrations(ctx, database, logger.InfoLogger)
	if err != nil {
		logger.Error("Database migration failed: %v", err)
		os.Exit(1)
	}
	logger.Info("Applied %v database migrations.", count)

	if err := fixture.CreateFixtures(cfg.FixtureFile, database, logger.InfoLogger); err != nil {
		logger.Error("Failed to create fixtures: %v", err)
		os.Exit(1)
	}
	logger.Info("Successfully created fixtures.")

	if err := api.StartServer(ctx, cfg, database, logger.InfoLogger); err != nil {
		logger.Error("HTTP server error: %v", err)
		os.Exit(1)
	}
}

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
	"goq/internal/logstore"
	_ "goq/migrations"
)

func main() {
	
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.LoadConfig()

	log, err := logger.Init(cfg)
	if err != nil {
		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer log.Close()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Error("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.RunMigrations(ctx, database, log); err != nil {
		log.Error("Database migration failed: %v", err)
		os.Exit(1)
	}

	if err := fixture.CreateFixtures(ctx, cfg, database, log); err != nil {
		log.Error("Failed to create fixtures: %v", err)
		os.Exit(1)
	}

	logStore, err := logstore.Init(ctx, cfg, database, log)
	if err != nil {
		log.Error("Failed to initialize message log store: %v", err)
		os.Exit(1)
	}
	defer logStore.Close()

	if err := api.StartServer(ctx, cfg, database, log, logStore); err != nil {
		log.Error("HTTP server error: %v", err)
		os.Exit(1)
	}
}

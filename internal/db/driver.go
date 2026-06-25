package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Connect(dsn string) (*sql.DB, error) {
	dir := filepath.Dir(dsn)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %q: %w", dir, err)
		}
	}

	absPath, _ := filepath.Abs(dsn)
	f, err := os.OpenFile(dsn, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("pre-flight check failed for %q (abs: %q): write permission denied: %w", dsn, absPath, err)
	}
	f.Close()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	_, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to configure sqlite PRAGMAs: %w", err)
	}

	return db, nil
}

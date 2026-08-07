package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/trhys/Recipe-Repo-2/internal/data"
)

func main() {
	dbUrl := os.Getenv("DB")
	if dbUrl == "" {
		slog.Error("Environment load failure", "missing", "DB")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		slog.Error("Failed to establish db connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx := context.Background()

	slog.Info("Running categories migration...")
	if err := data.SeedCategories(db, ctx); err != nil {
		slog.Error("Migration failure", "error", err)
		os.Exit(1)
	}
  
	slog.Info("Categories migration complete")
}

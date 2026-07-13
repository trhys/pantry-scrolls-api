package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/trhys/Recipe-Repo-2/internal/data"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	godotenv.Load()

	dbUrl := os.Getenv("DB")
	if dbUrl == "" {
		slog.Error("Environment load failure", "missing", "DB")
		os.Exit(1)
	}

	imagePlaceholder := os.Getenv("IMAGE_PLACEHOLDER")
	if imagePlaceholder == "" {
		slog.Error("Environment load failure", "missing", "IMAGE_PLACEHOLDER")
		os.Exit(1)
	}

	userpw := os.Getenv("USERPW")
	if userpw == "" {
		slog.Error("Environment load failure", "missing", "USERPW")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		slog.Error("Failed to establish db connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	if err := data.InitDBIngredients(imagePlaceholder, db, ctx); err != nil {
		slog.Error("Seed failure", "error", err)
		os.Exit(1)
	}

	hash, err := auth.HashPassword(userpw)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		os.Exit(1)
	}

	if err := data.InitDBRecipes(imagePlaceholder, db, ctx, hash); err != nil {
		slog.Error("Seed failure", "error", err)
		os.Exit(1)
	}

	slog.Info("Database seeding completed successfully")
}

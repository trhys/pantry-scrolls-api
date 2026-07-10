package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/trhys/Recipe-Repo-2/internal/data"
	"github.com/trhys/Recipe-Repo-2/internal/metrics"
	"github.com/trhys/Recipe-Repo-2/internal/server"
)

func main() {
	// initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := server.GetConfig()

	cfg.StartReaper()

	// Check database seeding
	if err := data.InitDBIngredients(cfg.ImagePlaceholder, cfg.DBConn, context.Background()); err != nil {
		slog.Error("Seed failure", "error", err)
		os.Exit(1)
	}

	hash, _ := auth.HashPassword(cfg.Root.Pass)

	if err := data.InitDBRecipes(cfg.ImagePlaceholder, cfg.DBConn, context.Background(), hash); err != nil {
		slog.Error("Seed failure", "error", err)
		os.Exit(1)
	}

	// Load server
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.React},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		AllowCredentials: true,
	})

	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)

	mux := server.GetRouter(cfg, reg)
	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: c.Handler(cfg.MetricsMiddleware(m)(mux)),
	}

	slog.Info("Successfully loaded server...")

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Listener crashed!", "error", err)
		os.Exit(1)
	}
}

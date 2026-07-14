package main

import (
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
	"github.com/trhys/Recipe-Repo-2/internal/metrics"
	"github.com/trhys/Recipe-Repo-2/internal/server"
)

func main() {
	// initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := server.GetConfig()

	cfg.StartReaper()

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

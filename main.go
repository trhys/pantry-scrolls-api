package main

import (
	"context"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/trhys/Recipe-Repo-2/internal/data"
	"github.com/trhys/Recipe-Repo-2/internal/server"
)

func main() {
	cfg := server.GetConfig()

	// Check database seeding
	if err := data.InitDBIngredients(cfg.ImagePlaceholder, cfg.DBConn, context.Background()); err != nil {
		log.Panic("Failed to seed database ingredients")
	}

	hash, _ := auth.HashPassword(cfg.Root.Pass)

	if err := data.InitDBRecipes(cfg.ImagePlaceholder, cfg.DBConn, context.Background(), hash); err != nil {
		log.Panic("Failed to seed database recipes")
	}

	// Load server
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.React},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		AllowCredentials: true,
	})

	mux := server.GetRouter(cfg)
	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: c.Handler(mux),
	}

	log.Print("Successfully loaded server...")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

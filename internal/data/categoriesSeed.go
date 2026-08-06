package data

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
  "fmt"
	"log"

	"github.com/lib/pq"
	pb "github.com/schollz/progressbar/v3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

//go:embed seedManifest.json
var manifest []byte

func SeedCategories(db *sql.DB, ctx context.Context) error {
	log.Println("Loading seed from JSON...")

	var seed struct {
		Ingredients []struct {
			Name        string `json:"name"`
      Category    string `json:"category"`
		} `json:"ingredients"`
	}

	if err := json.Unmarshal(manifest, &seed); err != nil {
		return fmt.Errorf("Failed to unmarshal JSON!")
	}

	log.Println("Successfully read file - seeding categories...")

	dbConn := database.New(db)
  defer dbConn.Close()

	bar := pb.Default(int64(len(seed.Ingredients)))
	for _, u := range seed.Ingredients {
		if _, err := dbConn.AddCategory(ctx, u.Name); err == nil {
			continue
		} else {
      log.Printf("ERROR FOR INGREDIENT %s --- %v", u.Name, err)
      continue
    }
  }

  log.Printf("Completed...")
  return nil
}

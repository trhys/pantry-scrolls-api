package data

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
  "fmt"
	"log"

	_ "github.com/lib/pq"
	pb "github.com/schollz/progressbar/v3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

//go:embed seedManifest.json
var manifestCategories []byte

func SeedCategories(db *sql.DB, ctx context.Context) error {
	log.Println("Loading seed from JSON...")

	var seed struct {
		Ingredients []struct {
			Name        string `json:"name"`
      Category    string `json:"category"`
		} `json:"ingredients"`
	}

	if err := json.Unmarshal(manifestCategories, &seed); err != nil {
      return fmt.Errorf("Failed to unmarshal JSON! ERROR: %v", err)
	}

	log.Println("Successfully read file - seeding categories...")

	dbConn := database.New(db)

	bar := pb.Default(int64(len(seed.Ingredients)))
	for _, u := range seed.Ingredients {
		if err := dbConn.AddCategory(ctx, database.AddCategoryParams{
          Name: u.Name,
          Category: u.Category,
        }); err == nil {
            bar.Add(1)
			continue
		} else {
          log.Printf("ERROR FOR INGREDIENT %s --- %v", u.Name, err)
          bar.Add(1)
          continue
    }
  }

  log.Printf("Completed...")
  return nil
}

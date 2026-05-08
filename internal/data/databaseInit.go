package data

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
	
	"github.com/lib/pq"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	pb "github.com/schollz/progressbar/v3"
)

//go:embed ingredientsManifest.json
var ingredientsManifest []byte

func InitDBIngredients(ik string, db *sql.DB, ctx context.Context) error {
	log.Println("Loading ingredients from JSON...")
	var ings struct {
		Ingredients []struct{
			Name string `json:"name"`
			Conversions []struct {
				From string `json:"from_unit"`
				To string `json:"to_unit"`
				Ratio float32 `json:"ratio"`
			} `json:"conversions"`
		} `json:"ingredients"`
	}

	if err := json.Unmarshal(ingredientsManifest, &ings); err != nil {
		log.Panic("Failed to unmarshal JSON!")
	}

	log.Println("Successfully read file - verifying entries...")

	dbConn := database.New(db)

	bar := pb.Default(int64(len(ings.Ingredients)))
	for _, i := range ings.Ingredients {
		ingID, err := dbConn.GetIngredientFromName(ctx, i.Name)
		if err == nil {
			c, err := dbConn.GetConversionsByID(ctx, ingID)
			if err == nil && len(c) == len(i.Conversions) {
				bar.Add(1)
				continue
			} else {
				for index, conv := range i.Conversions {
					queryB := database.CreateConversionParams{
						IngredientID: ingID,
						FromUnit: conv.From,
						ToUnit: conv.To,
						Ratio: conv.Ratio,
					}
				
					if err := dbConn.CreateConversion(ctx, queryB); err != nil {
						if pqErr, ok := err.(*pq.Error); ok {
							if pqErr.Code == "23505" {
								log.Printf("Entry exists: %s --- continuing...", queryB.IngredientID)
								continue
							}
						}
						log.Printf("Failed to create conversion at position: %d - %s", index, i.Name)
					}
				}
				bar.Add(1)
				continue
			}
		}

		queryA := database.CreateIngredientParams{
			Name: i.Name,
			ImageKey: ik,
		}

		ingredient, err := dbConn.CreateIngredient(ctx, queryA) 
		if err != nil {
			log.Printf("Failed on ingredient: %s - ERROR: %v", i.Name, err)
			log.Panic("Couldn't create ingredient during setup")
		}

		for index, conv := range i.Conversions {
			queryB := database.CreateConversionParams{
				IngredientID: ingredient.ID,
				FromUnit: conv.From,
				ToUnit: conv.To,
				Ratio: conv.Ratio,
			}

			if err := dbConn.CreateConversion(ctx, queryB); err != nil {
				log.Printf("Failed to create conversion at position: %d - %s", index, i.Name)
			}
		}
		bar.Add(1)
	}

	return nil
}

//go:embed recipesManifest.json
var recipesManifest []byte

func InitDBRecipes(ik string, db *sql.DB, ctx context.Context, userpw string) error {
	log.Println("Loading recipes from JSON...")
	var recipes struct {
		Recipes []struct{
			Title           string `json:"title"`
			Description     string `json:"description"`
			Ingredients     []struct{
				Name            string  `json:"name"`
				Quantity        float32 `json:"quantity"`
				Unit            string `json:"unit"`
			} `json:"ingredients"`
			Instructions    string `json:"instructions"`			} `json:"recipes"`
	}

	if err := json.Unmarshal(recipesManifest, &recipes); err != nil {
		log.Panic("Failed to unmarshal JSON!")
	}

	log.Println("Successfully read file - verifying entries...")

	dbConn := database.New(db)

	// Create/Verify user
	user, _ := dbConn.CreateUser(ctx, database.CreateUserParams{
		Email: "recipereporoot@admin.trr",
		HashedPw: userpw,
		Name: "Recipe Repo",
	})

	bar := pb.Default(int64(len(recipes.Recipes)))
	for _, r := range recipes.Recipes {
		_, err := dbConn.CheckIfSeeded(ctx, r.Description)
		if err == nil {
			bar.Add(1)
			continue
		}

		query := database.CreateRecipeParams{
			Title: r.Title,
			Author: user.Name,
			UserID: user.ID,
			Description: r.Description,
			ImageKey: ik,
			Instructions: r.Instructions,
		}

		rec, err := dbConn.CreateRecipe(ctx, query)
		if err != nil {
			log.Printf("Failed to create recipe: %s ERROR: %v", r.Title, err)
		}

		for _, ing := range r.Ingredients {
			id, err := dbConn.GetIngredientFromName(ctx, ing.Name)
			if err != nil {
				log.Printf("Failed to fetch ingredient ID during recipe creation - RECIPE: %s - ERROR: %v", r.Title, err)
				continue
			}

			query := database.AddToRecipeParams{
				RecipeID: rec.ID,
				IngredientID: id,
				Quantity: ing.Quantity,
				Unit: ing.Unit,
			}

			if _, err := dbConn.AddToRecipe(ctx, query); err != nil {
				log.Printf("Failed to add ingredient to recipe: %s Ingredient id: %s ERROR: %v", r.Title, id, err)
			}
		}

		bar.Add(1)
	}

	return nil
}

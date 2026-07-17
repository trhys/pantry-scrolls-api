package data

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
    "strings"

	"github.com/lib/pq"
	pb "github.com/schollz/progressbar/v3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

//go:embed seedManifest.json
var manifest []byte

func InitDBIngredients(ik string, db *sql.DB, ctx context.Context) error {
	log.Println("Loading seed from JSON...")

	var seed struct {
		Units []struct {
			Name string `json:"name"`
			Abbr string `json:"abbreviation"`
		} `json:"units"`
		Universal []struct {
			Name string `json:"name"`
		} `json:"universal_units"`
		Retail []struct {
			Name string `json:"name"`
		} `json:"retail_units"`
		RetailConversions []struct {
			UnivUnit string  `json:"universal_unit"`
			RetUnit  string  `json:"retail_unit"`
			Ratio    float32 `json:"ratio"`
		} `json:"retail_conversions"`
		Ingredients []struct {
			Name        string `json:"name"`
			Conversions []struct {
				From  string  `json:"from_unit"`
				To    string  `json:"to_unit"`
				Ratio float32 `json:"ratio"`
			} `json:"conversions"`
			RetailUnits []string `json:"retail_units"`
		} `json:"ingredients"`
	}

	if err := json.Unmarshal(manifest, &seed); err != nil {
		log.Panic("Failed to unmarshal JSON!")
	}

	log.Println("Successfully read file - verifying entries...")

	dbConn := database.New(db)

	bar := pb.Default(int64(len(seed.Units) + len(seed.Universal) + len(seed.Retail) + len(seed.RetailConversions)))
	for _, u := range seed.Units {
		if _, err := dbConn.GetUnit(ctx, u.Name); err == nil {
			continue
		}

		if err := dbConn.CreateUnit(ctx, database.CreateUnitParams{
			Name:         u.Name,
			Abbreviation: u.Abbr,
		}); err != nil {
			log.Printf("Failed to create unit: %v", err)
		}
		bar.Add(1)
	}

	for _, v := range seed.Universal {
		if _, err := dbConn.GetUniversalUnit(ctx, v.Name); err == nil {
			continue
		}

		if err := dbConn.CreateUniversalUnit(ctx, v.Name); err != nil {
			log.Printf("Failed to create universal unit: %v", err)
		}
		bar.Add(1)
	}

	for _, r := range seed.Retail {
		if _, err := dbConn.GetRetailUnit(ctx, r.Name); err == nil {
			continue
		}

		if err := dbConn.CreateRetailUnit(ctx, r.Name); err != nil {
			log.Printf("Failed to create retail unit: %v", err)
		}
		bar.Add(1)
	}

	for _, rc := range seed.RetailConversions {
		if _, err := dbConn.CheckRetailConversion(ctx, database.CheckRetailConversionParams{
			UniversalUnit: rc.UnivUnit,
			RetailUnit:    rc.RetUnit,
		}); err == nil {
			continue
		}

		if err := dbConn.CreateRetailConversion(ctx, database.CreateRetailConversionParams{
			UniversalUnit: rc.UnivUnit,
			RetailUnit:    rc.RetUnit,
			Ratio:         rc.Ratio,
		}); err != nil {
			log.Printf("Failed to create retail conversion: %v", err)
		}
		bar.Add(1)
	}

	bar = pb.Default(int64(len(seed.Ingredients)))
	for _, i := range seed.Ingredients {
		ingID, err := dbConn.GetIngredientFromName(ctx, i.Name)
		if err == nil {
			c, err := dbConn.GetConversionsByID(ctx, ingID)
			if err == nil && len(c) == len(i.Conversions) {
				bar.Add(1)
				continue
			} else {
				for index, conv := range i.Conversions {
					queryB := database.CreateUniversalConversionParams{
						IngredientID: ingID,
						FromUnit:     conv.From,
						ToUnit:       conv.To,
						Ratio:        conv.Ratio,
					}

					if err := dbConn.CreateUniversalConversion(ctx, queryB); err != nil {
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
			Name:     i.Name,
			ImageKey: ik,
		}

		ingredient, err := dbConn.CreateIngredient(ctx, queryA)
		if err != nil {
			log.Printf("Failed on ingredient: %s - ERROR: %v", i.Name, err)
			log.Panic("Couldn't create ingredient during setup")
		}

		for index, conv := range i.Conversions {
			queryB := database.CreateUniversalConversionParams{
				IngredientID: ingredient.ID,
				FromUnit:     conv.From,
				ToUnit:       conv.To,
				Ratio:        conv.Ratio,
			}

			if err := dbConn.CreateUniversalConversion(ctx, queryB); err != nil {
				log.Printf("Failed to create conversion at position: %d - %s", index, i.Name)
			}
		}

		for _, retail_unit := range i.RetailUnits {
			queryC := database.CreateIngredientRetailUnitParams{
				IngredientID: ingredient.ID,
				RetailUnit:   retail_unit,
			}

			if err := dbConn.CreateIngredientRetailUnit(ctx, queryC); err != nil {
				log.Printf("Failed to pair ingredient to retail unit: %v", err)
			}
		}

		bar.Add(1)
	}

	return nil
}

//go:embed recipesSeed.json
var recipesManifest []byte

//go:embed charactersManifest.json
var characters []byte

func InitDBRecipes(ik string, db *sql.DB, ctx context.Context, userpw string) error {
	log.Println("Loading recipes from JSON...")
	var recipes struct {
		Recipes []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Ingredients []struct {
				Name     string  `json:"name"`
				Quantity float32 `json:"quantity"`
				Unit     string  `json:"unit"`
			} `json:"ingredients"`
			Instructions string `json:"instructions"`
     Author string `json:"author"`
		} `json:"recipes"`
	}

	if err := json.Unmarshal(recipesManifest, &recipes); err != nil {
		log.Panic("Failed to unmarshal JSON!")
	}

	log.Println("Successfully read file - verifying entries...")

	dbConn := database.New(db)

  var chars struct {
        Characters []struct {
            Name string `json:"name"`
        } `json:"characters"`
  }

  if err := json.Unmarshal(characters, &chars); err != nil {
		log.Panic("Failed to unmarshal JSON!")
  }

  log.Println("Successfully read file - creating user profiles...")

  for _, char := range chars.Characters {
        name = strings.ToLower(strings.Trim(char.Name))
	    dbConn.CreateUser(ctx, database.CreateUserParams{
		    Email:    name + "@admin.trr",
		    HashedPw: userpw,
		    Name:     char.Name,
	    })
  }

	bar := pb.Default(int64(len(recipes.Recipes)))
	for _, r := range recipes.Recipes {
		_, err := dbConn.CheckIfSeeded(ctx, r.Description)
		if err == nil {
			bar.Add(1)
			continue
		}

   user, err := dbConn.GetUserByEmail(ctx, r.Author + "@admin.trr")
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return err
	}


		query := database.CreateRecipeParams{
			Title:        r.Title,
			Author:       user.Name,
			UserID:       user.ID,
			Description:  r.Description,
			ImageKey:     ik,
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
				RecipeID:     rec.ID,
				IngredientID: id,
				Quantity:     ing.Quantity,
				Unit:         ing.Unit,
			}

			if _, err := dbConn.AddToRecipe(ctx, query); err != nil {
				log.Printf("Failed to add ingredient to recipe: %s Ingredient id: %s ERROR: %v", r.Title, id, err)
			}
		}

		bar.Add(1)
	}

	return nil
}

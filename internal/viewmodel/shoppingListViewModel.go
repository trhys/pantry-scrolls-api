package viewmodel

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	db "github.com/trhys/Recipe-Repo-2/internal/database"
)

type ShoppingList struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ShoppingListViewModel struct {
	ShoppingList
	Recipes []Recipe `json:"recipes"`
}

type UserListsViewModel struct {
	UserLists []ShoppingList `json:"shopping_lists"`
}

type PrintViewModel struct {
	Name        string       `json:"name"`
	Ingredients []Ingredient `json:"items"`
}

// Display for viewing a single list with it's recipes
func (builder *VMFactory) GenerateShoppingListViewModel(list db.ShoppingList, recipes []db.GetRecipesFromListRow) ShoppingListViewModel {
	recipeViewmodel := builder.GenerateRecipeViewModel(recipes, nil)
	model := ShoppingListViewModel{
		ShoppingList: ShoppingList{
			ID:        list.ID,
			Name:      list.Name,
			CreatedAt: list.CreatedAt,
			UpdatedAt: list.UpdatedAt,
		},
		Recipes: recipeViewmodel.Recipes,
	}

	return model
}

// Display for all of the user's lists
func GenerateUserListsViewModel(lists []db.ShoppingList) UserListsViewModel {
	model := UserListsViewModel{
		UserLists: make([]ShoppingList, 0, len(lists)),
	}

	for _, list := range lists {
		model.UserLists = append(model.UserLists, ShoppingList{
			ID:        list.ID,
			Name:      list.Name,
			CreatedAt: list.CreatedAt,
			UpdatedAt: list.UpdatedAt,
		})
	}

	return model
}

// Display all the items(ingredients) on a list
func GeneratePrintViewModel(listName string, printout []db.PrintListRow, dbConn *db.Queries) PrintViewModel {
	model := PrintViewModel{
		Name: listName,
	}

	type agg struct {
		id             uuid.UUID
		universal_unit string
	}

	total := make(map[agg]struct {
		name     string
		quantity float32
	})

	for _, p := range printout {
		key := agg{id: p.IngredientID, universal_unit: p.ToUnit}
		conversion := p.Quantity * p.Ratio

		item := total[key]
		item.name = p.Name
		item.quantity += conversion
		total[key] = item
	}

	for key, item := range total {
		retailConversions, err := dbConn.GetRetailConversion(context.Background(), db.GetRetailConversionParams{
			IngredientID:  key.id,
			UniversalUnit: key.universal_unit,
		})
		if err != nil {
			log.Printf("Failed to get retail conversions during shopping list print out. Ingedient: %s - ERROR: %v", item.name, err)
			continue
		}

		bestUnit, bestQuantity := getBestFit(retailConversions, item.quantity)
        if bestUnit == "unknown" {
          continue
        }

		model.Ingredients = append(model.Ingredients, Ingredient{
			ID:       key.id,
			Name:     item.name,
			Quantity: bestQuantity,
			Unit:     bestUnit,
		})
	}

	return model
}

// Print list helper
func getBestFit(conversions []db.RetailConversion, quantity float32) (string, float32) {
	if len(conversions) == 0 {
		return "unknown", quantity
	}

	if len(conversions) == 1 {
		convQuantity := float32(math.Ceil(float64(quantity / conversions[0].Ratio)))
		return conversions[0].RetailUnit, convQuantity
	}

	var bestUnit string
	var bestQuantity float32
	min := float32(math.MaxFloat32)

	for _, conv := range conversions {
		packages := float32(math.Ceil(float64(quantity / conv.Ratio)))
		totalVolume := packages * conv.Ratio

		if totalVolume < min {
			min = totalVolume
			bestQuantity = packages
			bestUnit = conv.RetailUnit
		} else if totalVolume == min {
			if packages < bestQuantity || bestQuantity == 0 {
				bestQuantity = packages
				bestUnit = conv.RetailUnit
			}
		}
	}

	return bestUnit, bestQuantity
}

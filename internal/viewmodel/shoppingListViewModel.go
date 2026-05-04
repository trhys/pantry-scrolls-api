package viewmodel

import (
	"math"
	"time"

	"github.com/google/uuid"
	db "github.com/trhys/Recipe-Repo-2/internal/database"
)

type ShoppingList struct {
        ID              uuid.UUID       `json:"id"`
        Name            string          `json:"name"`
	CreatedAt       time.Time       `json:"created_at"`
        UpdatedAt       time.Time       `json:"updated_at"`
}

type ShoppingListViewModel struct {
	ShoppingList
	Recipes		[]RecipeOnList		`json:"recipes"`
	Quantity	map[uuid.UUID]int32	`json:"quantity"`
	
}

type UserListsViewModel struct {
	UserName	string		`json:"username"`
	UserLists	[]ShoppingList	`json:"shopping_lists"`
}

type PrintViewModel struct {
	Name            string          `json:"name"`
	Ingredients	[]Ingredient	`json:"items"`
}

func GenerateShoppingListViewModel(list db.ShoppingList, recipes []db.GetRecipesFromListRow) ShoppingListViewModel {
	model := ShoppingListViewModel {
		ShoppingList: ShoppingList {
			ID: list.ID,
			Name: list.Name,
			CreatedAt: list.CreatedAt,
			UpdatedAt: list.UpdatedAt,
		},
		Quantity: make(map[uuid.UUID]int32),
		Recipes: GetRecipesOnList(recipes),
	}
	
	return model
}

func GenerateUserListsViewModel(username string, lists []db.ShoppingList) UserListsViewModel {
	model := UserListsViewModel{
		UserName: username,
		UserLists: make([]ShoppingList, 0, len(lists)),
	}

	for _, list := range lists {
                model.UserLists = append(model.UserLists, ShoppingList{
                        ID: list.ID,
                        Name: list.Name,
                        CreatedAt: list.CreatedAt,
                        UpdatedAt: list.UpdatedAt,
                })
        }

	return model
}

func GeneratePrintViewModel(listName string, printout []db.PrintListRow) PrintViewModel {
	model := PrintViewModel{
		Name: listName,
	}

	for _, p := range printout {
		conversion := math.Ceil(float64(p.Quantity * p.Ratio))
		unit := p.ToUnit
		model.Ingredients = append(model.Ingredients, Ingredient{
			ID: p.IngredientID,
			Name: p.Name,
			Quantity: float32(conversion),
			Unit: unit,
		})
	}

		return model
}

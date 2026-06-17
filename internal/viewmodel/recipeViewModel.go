package viewmodel

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

type Recipe struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Recipe data pulled for content cards - excludes some values
type RecipeCard struct {
	Recipe
	ImageURL string    `json:"image_url"`
	UserID   uuid.UUID `json:"user_id"`
	Author   string    `json:"author"`
}

// Full data pull for the recipe
type RecipeFull struct {
	Recipe
	UserID       uuid.UUID    `json:"user_id"`
	Author       string       `json:"author"`
	Description  string       `json:"description"`
	ImageURL     string       `json:"image_url"`
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions string       `json:"instructions"`
}

// Data for a shopping list view
type RecipeOnList struct {
	Recipe
	UserID   uuid.UUID `json:"user_id"`
	Author   string    `json:"author"`
	Quantity int32     `json:"quantity"`
}

type RecipeCardViewModel struct {
	Recipes []RecipeCard `json:"recipes"`
}

func (builder *VMFactory) GenerateRecipeCardViewModel(recipes []database.Recipe) RecipeCardViewModel {
	model := RecipeCardViewModel{}
	for _, r := range recipes {
		model.Recipes = append(model.Recipes, RecipeCard{
			Recipe: Recipe{
				ID:        r.ID,
				Title:     r.Title,
				CreatedAt: r.CreatedAt,
				UpdatedAt: r.UpdatedAt,
			},
			UserID:   r.UserID,
			Author:   r.Author,
			ImageURL: fmt.Sprintf("%s/%s", builder.S3cdn, r.ImageKey),
		})
	}

	return model
}

func (builder *VMFactory) GenerateRecipeFullViewModel(r database.Recipe, i []Ingredient) RecipeFull {
	return RecipeFull{
		Recipe: Recipe{
			ID:        r.ID,
			Title:     r.Title,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		},
		UserID:       r.UserID,
		Author:       r.Author,
		Description:  r.Description,
		ImageURL:     fmt.Sprintf("%s/%s", builder.S3cdn, r.ImageKey),
		Ingredients:  i,
		Instructions: r.Instructions,
	}
}

func GetRecipesOnList(recipes []database.GetRecipesFromListRow) []RecipeOnList {
	recipeList := make([]RecipeOnList, 0, len(recipes))
	for _, r := range recipes {
		recipeList = append(recipeList, RecipeOnList{
			Recipe: Recipe{
				ID:        r.ID,
				Title:     r.Title,
				CreatedAt: r.CreatedAt,
				UpdatedAt: r.UpdatedAt,
			},
			UserID:   r.UserID,
			Author:   r.Author,
			Quantity: r.Quantity,
		})
	}

	return recipeList
}

func (builder *VMFactory) GetRecipesForUser(recipes []database.Recipe) []RecipeCard {
	recipeList := make([]RecipeCard, 0, len(recipes))
	for _, r := range recipes {
		recipeList = append(recipeList, RecipeCard{
			Recipe: Recipe{
				ID:        r.ID,
				Title:     r.Title,
				CreatedAt: r.CreatedAt,
				UpdatedAt: r.UpdatedAt,
			},
			UserID:   r.UserID,
			Author:   r.Author,
			ImageURL: fmt.Sprintf("%s/%s", builder.S3cdn, r.ImageKey),
		})
	}

	return recipeList
}

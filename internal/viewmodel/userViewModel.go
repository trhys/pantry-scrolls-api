package viewmodel

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	ImageURL string    `json:"image_url"`
}

type PrivateUserViewModel struct {
	User
	Email         string         `json:"email"`
	CreatedAt     time.Time      `json:"created_at"`
	Recipes       []Recipe       `json:"recipes"`
	ShoppingLists []ShoppingList `json:"shopping_lists"`
}

type PublicUserViewModel struct {
	User
	Recipes []Recipe `json:"recipes"`
}

type SessionViewModel struct {
	User
	Email string `json:"email"`
	JWT   string `json:"token"`
	RT    string `json:"refresh_token"`
}

type RefreshViewModel struct {
	User
	Email string `json:"email"`
}

func (builder *VMFactory) GeneratePrivateUser(user database.GetUserRow, recipes any) PrivateUserViewModel {
	recipeViewmodel := builder.GenerateRecipeViewModel(recipes, nil)
	model := PrivateUserViewModel{
		User: User{
			ID:       user.ID,
			Name:     user.Name,
			ImageURL: fmt.Sprintf("%s/%s", builder.S3cdn, user.ImageKey),
		},
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		Recipes:   recipeViewmodel.Recipes,
	}

	lists, err := builder.DB.GetUserLists(context.Background(), user.ID)
	if err != nil {
		log.Printf("Failed to get user lists for private user viewmodel: USER: %s ERROR: %v", user.ID, err)
		return model
	}

	model.ShoppingLists = GenerateUserListsViewModel(lists).UserLists

	return model
}

func (builder *VMFactory) GeneratePublicUser(user database.GetUserRow, recipes any) PublicUserViewModel {
	recipeViewmodel := builder.GenerateRecipeViewModel(recipes, nil)
	model := PublicUserViewModel{
		User: User{
			ID:       user.ID,
			Name:     user.Name,
			ImageURL: fmt.Sprintf("%s/%s", builder.S3cdn, user.ImageKey),
		},
		Recipes: recipeViewmodel.Recipes,
	}

	return model
}

func (builder *VMFactory) GenerateSession(user database.GetUserHashRow, token, refreshToken string) SessionViewModel {
	return SessionViewModel{
		User: User{
			ID:       user.ID,
			Name:     user.Name,
			ImageURL: fmt.Sprintf("%s/%s", builder.S3cdn, user.ImageKey),
		},
		Email: user.Email,
		JWT:   token,
		RT:    refreshToken,
	}
}

func (builder *VMFactory) RefreshSession(user database.RefreshUserRow) RefreshViewModel {
	return RefreshViewModel{
		User: User{
			ID:       user.ID,
			Name:     user.Name,
			ImageURL: fmt.Sprintf("%s/%s", builder.S3cdn, user.ImageKey),
		},
		Email: user.Email,
	}
}

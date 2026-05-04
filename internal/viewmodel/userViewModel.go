package viewmodel

import (
	"time"

	"github.com/google/uuid"
        "github.com/trhys/Recipe-Repo-2/internal/database"
)

type User struct{
	ID              uuid.UUID 	`json:"id"`
        Name            string 		`json:"name"`
}

type PrivateUserViewModel struct{
	User
        Email           string 			`json:"email"`
        CreatedAt       time.Time		`json:"created_at"`
	Recipes 	[]RecipeCard 		`json:"recipes"`
}

type PublicUserViewModel struct{
	User
	Recipes 	[]RecipeCard 	`json:"recipes"`
}

type SessionViewModel struct{
	User
	Email	string	`json:"email"`
	JWT	string	`json:"token"`
	RT	string	`json:"refresh_token"`
}

func (builder *VMFactory) GeneratePrivateUser(user database.GetUserRow, recipes []database.Recipe) PrivateUserViewModel {
	model := PrivateUserViewModel{
		User: User{
			ID:	user.ID,
			Name:	user.Name,
		},
		Email:		user.Email,
		CreatedAt:	user.CreatedAt,
		Recipes:	builder.GetRecipesForUser(recipes),
	}

	return model
}

func (builder *VMFactory) GeneratePublicUser(user database.GetUserRow, recipes []database.Recipe) PublicUserViewModel {
	model := PublicUserViewModel{
		User: User{
			ID:	user.ID,
			Name:	user.Name,
		},
		Recipes:	builder.GetRecipesForUser(recipes),
	}

	return model
}

func GenerateSession(user database.GetUserHashRow, token, refreshToken string) SessionViewModel {
	return SessionViewModel{
		User: User{
			ID:	user.ID,
			Name:	user.Name,
		},
		Email:	user.Email,
		JWT:	token,
		RT:	refreshToken,
	}
}

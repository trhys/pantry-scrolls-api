package main

import (
	"fmt"
	"html/template"
	"path/filepath"
	"net/http"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
	util "github.com/trhys/Recipe-Repo-2/internal/utility"
)

// Create a new, empty shopping list
func (cfg *apiConfig) handlerCreateShoppingList(w http.ResponseWriter, r *http.Request) {
	var req struct{
		Name	string `json:"name"`
	}

	// AUTH
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
        if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Decode request body
	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(w, 400, "Bad request", fmt.Errorf("Failed to decode request: ERROR: %v", err))
		return
	}

	// Create list
	list, err := cfg.db.CreateShoppingList(r.Context(), database.CreateShoppingListParams{
		Name: req.Name,
		UserID: requesterID,
	})

	if err != nil {
		respondFail(w, 500, "Database error", fmt.Errorf("Failed to perform CreateShoppingList query: %v", err))
		return
	}

	respondJSON(w, 200, viewmodel.ShoppingList{
		ID: list.ID,
		Name: list.Name,
		CreatedAt: list.CreatedAt,
		UpdatedAt: list.UpdatedAt,
	})
}

// Add recipe to shopping list
func (cfg *apiConfig) handlerAddToShoppingList(w http.ResponseWriter, r *http.Request) {
	var req struct{
		ShoppingListID	uuid.UUID `json:"shopping_list_id"`
		RecipeID	uuid.UUID `json:"recipe_id"`
		Quantity	int32	  `json:"quantity"`
	}

	// AUTH
        requesterID, ok := r.Context().Value("userID").(uuid.UUID)
        if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Decode body
	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(w, 400, "Something went wrong", fmt.Errorf("Failed to decode request: ERROR: %v", err))
		return
	}

	// Link recipe to list by ID
	if err := cfg.db.AddRecipeToList(r.Context(), database.AddRecipeToListParams{
		ShoppingListID: req.ShoppingListID,
		RecipeID: req.RecipeID,
		Quantity: req.Quantity,
	}); err != nil {
		respondFail(w, 500, "Database error", fmt.Errorf("Failed to perform AddRecipeToList query: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

// Get shopping list by ID
func (cfg *apiConfig) handlerGetShoppingList(w http.ResponseWriter, r *http.Request) {
	val := r.PathValue("shopping_list_id")
	listID, err := uuid.Parse(val) 
	if err != nil {
		respondFail(w, 404, "invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	// AUTH
        requesterID, ok := r.Context().Value("userID").(uuid.UUID)
        if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	shoppingList, err := cfg.db.GetShoppingList(r.Context(), listID)
	if err != nil {
		respondFail(w, 404, "Couldn't find shopping list", fmt.Errorf("Failed to find shopping list with ID: %s, ERROR: %V", val, err))
		return
	}

	if requesterID != shoppingList.UserID {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Get recipes from list
	shoppingListRecipes, err := cfg.db.GetRecipesFromList(r.Context(), shoppingList.ID)
	if err != nil {
		respondFail(w, 404, "couldnt find recipes from list", fmt.Errorf("Failed to get recipes from shopping list id: %s, ERROR: %v", val, err))
		return
	}

	model := viewmodel.GenerateShoppingListViewModel(shoppingList, shoppingListRecipes)	

	if r.Header.Get("Accept") == "application/json" {
		respondJSON(w, 200, model)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join("app", "templates", "shopping_list.html"))
        if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to render HTML template: %v", err))
                return
        }

        tmpl.Execute(w, model)
}

// List the user's shopping lists
func (cfg *apiConfig) handlerGetUsersShoppingLists(w http.ResponseWriter, r *http.Request) {
	val := r.PathValue("user_id")
	id, err := uuid.Parse(val)
        if err != nil {
		respondFail(w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
                return
        }

        user, err := cfg.db.GetUser(r.Context(), id)
        if err != nil {
		respondFail(w, 404, "Couldn't find user", fmt.Errorf("Failed to find user with ID: %s ERROR: %v", val, err))
                return
        }

	// Authorization
        _, ok := r.Context().Value("userID").(uuid.UUID)
        if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", val))
                return
        }

	// Get lists
	lists, err := cfg.db.GetUserLists(r.Context(), user.ID)
	if err != nil {
		respondFail(w, 404, "Couldn't retrieve user's shopping lists", fmt.Errorf("Failed to get lists from database: %v", err))
		return
	}

	model := viewmodel.GenerateUserListsViewModel(user.Name, lists)

	if r.Header.Get("Accept") == "application/json" {
                respondJSON(w, 200, model)
                return
        }

        tmpl, err := template.ParseFiles(filepath.Join("app", "templates", "user_shopping_lists.html"))
        if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to render HTML template: %v", err))
                return
        }

        tmpl.Execute(w, model)
}

// Print the shopping lists ingredients in converted retail units
func (cfg *apiConfig) handlerPrintList(w http.ResponseWriter, r *http.Request) {
	// AUTH
        requesterID, ok := r.Context().Value("userID").(uuid.UUID)
        if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Get list ID
	val := r.PathValue("shopping_list_id")
	id, err := uuid.Parse(val)
        if err != nil {
		respondFail(w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
                return
        }

	// Verify ownership against JWT subject
	list, err := cfg.db.GetListOwner(r.Context(), id)
	if err != nil || list.UserID != requesterID {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get list
	printed, err := cfg.db.PrintList(r.Context(), id)
	if err != nil {
		respondFail(w, 404, "Couldn't locate list", fmt.Errorf("Failed to get list from database: %v", err))
		return
	}

	model := viewmodel.GeneratePrintViewModel(list.Name, printed)

	if r.Header.Get("Accept") == "application/json" {
		respondJSON(w, 200, model)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join("app", "templates", "print_list.html"))
        if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to render HTML template: %v", err))
                return
        }

        tmpl.Execute(w, model)
}

// Delete shopping list
func (cfg *apiConfig) handlerDeleteShoppingList(w http.ResponseWriter, r *http.Request) {
	// Get list ID
	val := r.PathValue("shopping_list_id")
	id, err := uuid.Parse(val)
        if err != nil {
		respondFail(w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
                return
        }

	// AUTH
        requesterID, ok := r.Context().Value("userID").(uuid.UUID)
        if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Verify ownership against JWT subject
	list, err := cfg.db.GetListOwner(r.Context(), id)
	if err != nil || list.UserID != requesterID {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Authorized - delete list
	if err := cfg.db.DeleteShoppingList(r.Context(), id); err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to delete list from database: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

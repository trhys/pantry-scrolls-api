package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	util "github.com/trhys/Recipe-Repo-2/internal/utility"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

// Create a new, empty shopping list
func (cfg *ApiConfig) handlerCreateShoppingList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	// AUTH
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Decode request body
	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Failed to decode request: ERROR: %v", err))
		return
	}

	// validate input
	if strings.TrimSpace(req.Name) == "" {
		respondFail(r, w, 400, "Invalid list name", fmt.Errorf("List name must not be empty - list name: %s", req.Name))
		return
	}

	// Create list
	list, err := cfg.DB.CreateShoppingList(r.Context(), database.CreateShoppingListParams{
		Name:   req.Name,
		UserID: requesterID,
	})

	if err != nil {
		respondFail(r, w, 500, "Database error", fmt.Errorf("Failed to perform CreateShoppingList query: %v", err))
		return
	}

	respondJSON(w, 200, viewmodel.ShoppingList{
		ID:        list.ID,
		Name:      list.Name,
		CreatedAt: list.CreatedAt,
		UpdatedAt: list.UpdatedAt,
	})
}

// Add recipe to shopping list
func (cfg *ApiConfig) handlerAddToShoppingList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecipeID uuid.UUID `json:"recipe_id"`
		Quantity int32     `json:"quantity"`
	}

	// AUTH
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get list id
	val := r.PathValue("shopping_list_id")
	id, err := uuid.Parse(val)
	if err != nil {
		respondFail(r, w, 404, "Invalid uuid", fmt.Errorf("Failed to parse uuid from path: %v", err))
		return
	}

	// Decode body
	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(r, w, 400, "Something went wrong", fmt.Errorf("Failed to decode request: ERROR: %v", err))
		return
	}

	// validate input
	if req.Quantity <= 0 {
		respondFail(r, w, 400, "Bad quantity", fmt.Errorf("(AddToList) Quantity must be greater than 0 - got: %d", req.Quantity))
		return
	}

	// Link recipe to list by ID
	if err := cfg.DB.AddRecipeToList(r.Context(), database.AddRecipeToListParams{
		ShoppingListID: id,
		RecipeID:       req.RecipeID,
		Quantity:       req.Quantity,
	}); err != nil {
		if err.(*pq.Error).Code == "23505" {
			if err := cfg.DB.UpdateShoppingListRecipe(r.Context(), database.UpdateShoppingListRecipeParams{
				ShoppingListID: id,
				RecipeID:       req.RecipeID,
				Quantity:       req.Quantity,
			}); err != nil {
				respondFail(r, w, 500, "Database error", fmt.Errorf("Failed to perform AddRecipeToList query: %v", err))
				return
			}
		} else {
			respondFail(r, w, 404, "Couldn't add recipe", fmt.Errorf("Failed to add recipe to list: %v", err))
			return
		}
	}

	respondJSON(w, 204, nil)
}

// Get shopping list by ID
func (cfg *ApiConfig) handlerGetShoppingList(w http.ResponseWriter, r *http.Request) {
	val := r.PathValue("shopping_list_id")
	listID, err := uuid.Parse(val)
	if err != nil {
		respondFail(r, w, 404, "invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	// AUTH
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	shoppingList, err := cfg.DB.GetShoppingList(r.Context(), listID)
	if err != nil {
		respondFail(r, w, 404, "Couldn't find shopping list", fmt.Errorf("Failed to find shopping list with ID: %s, ERROR: %V", val, err))
		return
	}

	if requesterID != shoppingList.UserID {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get recipes from list
	shoppingListRecipes, err := cfg.DB.GetRecipesFromList(r.Context(), shoppingList.ID)
	if err != nil {
		respondFail(r, w, 404, "couldnt find recipes from list", fmt.Errorf("Failed to get recipes from shopping list id: %s, ERROR: %v", val, err))
		return
	}

	model := cfg.Vmf.GenerateShoppingListViewModel(shoppingList, shoppingListRecipes)

	respondJSON(w, 200, model)
}

// List the user's shopping lists
func (cfg *ApiConfig) handlerGetUsersShoppingLists(w http.ResponseWriter, r *http.Request) {
	// Authorization
	id, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", id))
		return
	}

	user, err := cfg.DB.GetUser(r.Context(), id)
	if err != nil {
		respondFail(r, w, 404, "Couldn't find user", fmt.Errorf("Failed to find user with ID: %s ERROR: %v", id, err))
		return
	}

	// Get lists
	lists, err := cfg.DB.GetUserLists(r.Context(), user.ID)
	if err != nil {
		respondFail(r, w, 404, "Couldn't retrieve user's shopping lists", fmt.Errorf("Failed to get lists from database: %v", err))
		return
	}

	model := viewmodel.GenerateUserListsViewModel(lists)

	respondJSON(w, 200, model)
}

// Print the shopping lists ingredients in converted retail units
func (cfg *ApiConfig) handlerPrintList(w http.ResponseWriter, r *http.Request) {
	// AUTH
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get list ID
	val := r.PathValue("shopping_list_id")
	id, err := uuid.Parse(val)
	if err != nil {
		respondFail(r, w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	// Verify ownership against JWT subject
	list, err := cfg.DB.GetListOwner(r.Context(), id)
	if err != nil || list.UserID != requesterID {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get list
	printed, err := cfg.DB.PrintList(r.Context(), id)
	if err != nil {
		respondFail(r, w, 404, "Couldn't locate list", fmt.Errorf("Failed to get list from database: %v", err))
		return
	}

	model := viewmodel.GeneratePrintViewModel(list.Name, printed, cfg.DB)

	respondJSON(w, 200, model)
}

// Delete shopping list
func (cfg *ApiConfig) handlerDeleteShoppingList(w http.ResponseWriter, r *http.Request) {
	// Get list ID
	val := r.PathValue("shopping_list_id")
	id, err := uuid.Parse(val)
	if err != nil {
		respondFail(r, w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	// AUTH
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Verify ownership against JWT subject
	list, err := cfg.DB.GetListOwner(r.Context(), id)
	if err != nil || list.UserID != requesterID {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Authorized - delete list
	if err := cfg.DB.DeleteShoppingList(r.Context(), id); err != nil {
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Failed to delete list from database: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

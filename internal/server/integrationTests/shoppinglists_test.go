package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestCRUDShoppingList(t *testing.T) {
	cfg := server.GetConfig()
	tx, err := cfg.DBConn.Begin()
	if err != nil {
		t.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	cfg.DB = cfg.DB.WithTx(tx)

	mockSES := &MockSESClient{}
	cfg.SESClient = mockSES

	router := server.GetRouter(cfg)

	testUser := struct {
		input []byte
	}{
		input: []byte(`{"email":"tim@test.com", "password": "password", "name": "tim"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("Failed to create user for test")
	}

	// Login test user
	login := []byte(`{"email":"tim@test.com", "password": "password"}`)
	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("Failed to login test user")
	}

	var user vm.SessionViewModel
	if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	// Get session cookies
	response := w.Result()
	cookies := response.Cookies()
	var jwt *http.Cookie
	var rt *http.Cookie
	for _, c := range cookies {
		if c.Name == "jwt" {
			jwt = c
		} else if c.Name == "refresh_token" {
			rt = c
		}
	}

	// Pre-seed a test recipe to link against downstream shopping list tests
	recipe, err := cfg.DB.CreateRecipe(context.Background(), database.CreateRecipeParams{
		Title:        "Classic Spaghetti Bolognese",
		Author:       user.Name,
		UserID:       user.ID,
		Description:  "test",
		Instructions: "test",
		ImageKey:     "placeholder-key.png",
	})
	if err != nil {
		t.Fatalf("Failed to seed prerequisite test recipe: %v", err)
	}

	var testListID uuid.UUID

	// Create Shopping List
	t.Run("create shopping list", func(t *testing.T) {
		body := []byte(`{"name": "Weekly Grocery Run"}`)
		req := httptest.NewRequest("POST", "/api/shoppinglists", bytes.NewBuffer(body))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to create shopping list: got status %d", w.Code)
			return
		}

		responseResult := w.Result()
		defer responseResult.Body.Close()

		var responseBody vm.ShoppingList
		decoder := json.NewDecoder(responseResult.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&responseBody); err != nil {
			t.Errorf("Response structural validation failed: %v", err)
		}

		if responseBody.Name != "Weekly Grocery Run" {
			t.Errorf("Expected list name 'Weekly Grocery Run' got %s", responseBody.Name)
		}

		testListID = responseBody.ID
	})

	// Add Recipe to Shopping List
	t.Run("add recipe to list", func(t *testing.T) {
		bodyPayload := struct {
			RecipeID uuid.UUID `json:"recipe_id"`
			Quantity int32     `json:"quantity"`
		}{
			RecipeID: recipe.ID,
			Quantity: 2,
		}
		data, _ := json.Marshal(bodyPayload)

		url := "/api/shoppinglists/" + testListID.String()
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(data))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Expected 204 No Content linking recipe to list, got %d", w.Code)
		}
	})

	// Get Shopping List with Recipes Expanded
	t.Run("get shopping list by id", func(t *testing.T) {
		url := "/api/shoppinglists/" + testListID.String()
		req := httptest.NewRequest("GET", url, nil)
		req.SetPathValue("shopping_list_id", testListID.String())

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to retrieve shopping list: got status %d", w.Code)
			return
		}

		responseResult := w.Result()
		defer responseResult.Body.Close()

		var responseBody vm.ShoppingListViewModel
		decoder := json.NewDecoder(responseResult.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&responseBody); err != nil {
			t.Errorf("Response structural validation failed on extended list: %v", err)
		}
	})

	// List User's Shopping Lists
	t.Run("get user shopping lists", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shoppinglists", nil)

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to fetch user shopping lists: got status %d", w.Code)
			return
		}

		responseResult := w.Result()
		defer responseResult.Body.Close()

		var responseBody vm.UserListsViewModel
		decoder := json.NewDecoder(responseResult.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&responseBody); err != nil {
			t.Errorf("Response validation failed on user lists dashboard: %v", err)
		}
	})

	// Delete Shopping List
	t.Run("delete shopping list", func(t *testing.T) {
		url := "/api/shoppinglists/" + testListID.String()
		req := httptest.NewRequest("DELETE", url, nil)
		req.SetPathValue("shopping_list_id", testListID.String())

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Expected 204 No Content deleting shopping list, got %d", w.Code)
		}
	})
}

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestShoppingListEdgeCases(t *testing.T) {
	cfg := server.GetConfig()
	tx, err := cfg.DBConn.Begin()
	if err != nil {
		t.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	cfg.DB = cfg.DB.WithTx(tx)

	mockSES := &MockSESClient{}
	cfg.SESClient = mockSES

	testReg := prometheus.NewRegistry()
	router := server.GetRouter(cfg, testReg)

	// Create test user
	testUser := struct {
		input []byte
	}{
		input: []byte(`{"email":"list@test.com", "password": "password", "name": "list user"}`),
	}

	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Login
	login := []byte(`{"email":"list@test.com", "password": "password"}`)
	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var user vm.SessionViewModel
	json.NewDecoder(w.Body).Decode(&user)

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

	t.Run("get nonexistent shopping list", func(t *testing.T) {
		fakeListID := uuid.New()
		url := "/api/shoppinglists/" + fakeListID.String()

		req := httptest.NewRequest("GET", url, nil)
		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for nonexistent list: got status %d", w.Code)
		}
	})

	t.Run("get shopping list without auth", func(t *testing.T) {
		fakeListID := uuid.New()
		url := "/api/shoppinglists/" + fakeListID.String()

		req := httptest.NewRequest("GET", url, nil)
		// No auth cookies

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail auth check
		if w.Code >= 200 && w.Code < 300 {
			t.Errorf("Expected error for unauthenticated access: got status %d", w.Code)
		}
	})

	t.Run("create shopping list with invalid name", func(t *testing.T) {
		body := []byte(`{"name": ""}`) // Empty name
		req := httptest.NewRequest("POST", "/api/shoppinglists", bytes.NewBuffer(body))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == 200 || w.Code == 201 {
			t.Errorf("Expected error for empty name: got status %d", w.Code)
		}
	})

	t.Run("add invalid recipe to shopping list", func(t *testing.T) {
		// Create a shopping list first
		body := []byte(`{"name": "Test List"}`)
		req := httptest.NewRequest("POST", "/api/shoppinglists", bytes.NewBuffer(body))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var shoppingList vm.ShoppingList
		json.NewDecoder(w.Body).Decode(&shoppingList)

		// Try to add nonexistent recipe
		fakeRecipeID := uuid.New()
		addBody := []byte(`{"recipe_id":"` + fakeRecipeID.String() + `","quantity":1}`)
		url := "/api/shoppinglists/" + shoppingList.ID.String()

		req = httptest.NewRequest("POST", url, bytes.NewBuffer(addBody))
		ctx = context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		
		_, err := tx.Exec("SAVEPOINT add_invalid_recipe_sp")
		if err != nil {
			t.Fatalf("Failed to create savepoint: %v", err)
		}

		router.ServeHTTP(w, req)

		_, err = tx.Exec("ROLLBACK TO SAVEPOINT add_invalid_recipe_sp")
		if err != nil {
			t.Fatalf("Failed to rollback to savepoint: %v", err)
		}

		// Should fail since recipe doesn't exist
		if w.Code >= 200 && w.Code < 300 {
			t.Errorf("Expected error for invalid recipe: got status %d", w.Code)
		}
	})

	t.Run("add recipe with invalid quantity", func(t *testing.T) {
		// Create shopping list
		body := []byte(`{"name": "Test List 2"}`)
		req := httptest.NewRequest("POST", "/api/shoppinglists", bytes.NewBuffer(body))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var shoppingList vm.ShoppingList
		json.NewDecoder(w.Body).Decode(&shoppingList)

		// Create a recipe
		recipe, err := cfg.DB.CreateRecipe(context.Background(), database.CreateRecipeParams{
			Title:        "Test Recipe",
			Author:       user.Name,
			UserID:       user.ID,
			Description:  "test",
			Instructions: "test",
			ImageKey:     "placeholder-key.png",
		})
		if err != nil {
			t.Fatalf("Failed to create recipe: %v", err)
		}

		// Try to add with negative quantity
		addBody := []byte(`{"recipe_id":"` + recipe.ID.String() + `","quantity":-1}`)
		url := "/api/shoppinglists/" + shoppingList.ID.String()

		req = httptest.NewRequest("POST", url, bytes.NewBuffer(addBody))
		ctx = context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == 200 || w.Code == 204 {
			t.Errorf("Expected error for negative quantity: got status %d", w.Code)
		}
	})

	t.Run("access other user's shopping list", func(t *testing.T) {
		// Create second user
		testUser2 := struct {
			input []byte
		}{
			input: []byte(`{"email":"list2@test.com", "password": "password", "name": "list user 2"}`),
		}

		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser2.input))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Login user 2
		login2 := []byte(`{"email":"list2@test.com", "password": "password"}`)
		req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login2))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var user2 vm.SessionViewModel
		json.NewDecoder(w.Body).Decode(&user2)

		response := w.Result()
		cookies := response.Cookies()
		//var jwt2 *http.Cookie
		//var rt2 *http.Cookie
		for _, c := range cookies {
			if c.Name == "jwt" {
				//jwt2 = c
			} else if c.Name == "refresh_token" {
				//rt2 = c
			}
		}

		// Create shopping list for user1
		list1Body := []byte(`{"name": "User1 List"}`)
		req = httptest.NewRequest("POST", "/api/shoppinglists", bytes.NewBuffer(list1Body))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var list1 vm.ShoppingList
		json.NewDecoder(w.Body).Decode(&list1)

		// Try to access user1's list with user2's auth
		url := "/api/shoppinglists/" + list1.ID.String()
		req = httptest.NewRequest("GET", url, nil)

		ctx = context.WithValue(req.Context(), "userID", user2.ID)
		req = req.WithContext(ctx)
		//req.AddCookie(jwt2)
		//req.AddCookie(rt2)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("Expected 401 for unauthorized access: got status %d", w.Code)
		}
	})

	t.Run("delete shopping list", func(t *testing.T) {
		// Create a shopping list
		body := []byte(`{"name": "Delete Test"}`)
		req := httptest.NewRequest("POST", "/api/shoppinglists", bytes.NewBuffer(body))

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var shoppingList vm.ShoppingList
		json.NewDecoder(w.Body).Decode(&shoppingList)

		// Delete it
		url := "/api/shoppinglists/" + shoppingList.ID.String()
		req = httptest.NewRequest("DELETE", url, nil)

		ctx = context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)
		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 204 {
			t.Errorf("Failed to delete shopping list: got status %d", w.Code)
		}

		// Verify deletion
		req = httptest.NewRequest("GET", url, nil)
		ctx = context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 after deletion: got status %d", w.Code)
		}
	})
}

package server_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestGetIngredients(t *testing.T) {
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

	t.Run("get ingredient base", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/ingredients", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Failed to get ingredients: got status %d", w.Code)
			return
		}

		var response struct {
			Ingredients []struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			} `json:"ingredients"`
		}

		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
			return
		}

		if len(response.Ingredients) == 0 {
			t.Errorf("Expected ingredients in response")
		}
	})
}

func TestGetIngredientUnits(t *testing.T) {
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

	t.Run("get units for valid ingredient", func(t *testing.T) {
		// Get an ingredient first
		ingredients, err := cfg.DB.GetIngredients(context.Background())
		if err != nil || len(ingredients) == 0 {
			t.Fatalf("Failed to get ingredients for test: %v", err)
		}

		ingredientID := ingredients[0].ID
		url := "/api/ingredients/" + ingredientID.String() + "/units"

		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Failed to get units: got status %d", w.Code)
			return
		}

		var response vm.UnitsViewModel
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if len(response.Units) == 0 {
			t.Errorf("Expected units in response")
		}
	})

	t.Run("get units with invalid ingredient id", func(t *testing.T) {
		url := "/api/ingredients/invalid-uuid/units"

		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for invalid UUID: got status %d", w.Code)
		}
	})

	t.Run("get units with nonexistent ingredient", func(t *testing.T) {
		fakeID := uuid.New()
		url := "/api/ingredients/" + fakeID.String() + "/units"

		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for nonexistent ingredient: got status %d", w.Code)
		}
	})
}

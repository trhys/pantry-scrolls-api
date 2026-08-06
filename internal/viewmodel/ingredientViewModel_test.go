package viewmodel

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

func TestGenerateIngredientsWithConversionsViewModel(t *testing.T) {
	ingID := uuid.New()
	ingredients := []database.GetIngredientListRow{
		{
			IngredientID: ingID,
			Name:         "Salt",
			Quantity:     1.5,
			Unit:         "Teaspoon",
		},
	}
	conversions := map[uuid.UUID][]database.Conversion{
		ingID: {
			{IngredientID: ingID, FromUnit: "Teaspoon", ToUnit: "Tablespoon", Ratio: 0.333},
			{IngredientID: ingID, FromUnit: "Teaspoon", ToUnit: "Cup", Ratio: 0.0208},
		},
	}

	result := GenerateIngredientsWithConversionsViewModel(ingredients, conversions)

	if len(result) != 1 {
		t.Fatalf("expected 1 ingredient, got %d", len(result))
	}

	ing := result[0]
	if ing.ID != ingID {
		t.Errorf("expected ingredient ID %v, got %v", ingID, ing.ID)
	}
	if ing.Name != "Salt" {
		t.Errorf("expected Name 'Salt', got '%s'", ing.Name)
	}
	if ing.Quantity != 1.5 {
		t.Errorf("expected Quantity 1.5, got %f", ing.Quantity)
	}
	if ing.Unit != "Teaspoon" {
		t.Errorf("expected Unit 'Teaspoon', got '%s'", ing.Unit)
	}
	if len(ing.Conversions) != 2 {
		t.Fatalf("expected 2 conversions, got %d", len(ing.Conversions))
	}
	if ing.Conversions[0].FromUnit != "Teaspoon" {
		t.Errorf("expected FromUnit 'Teaspoon', got '%s'", ing.Conversions[0].FromUnit)
	}

	// ensure conversions appear in JSON output
	body, err := json.Marshal(ing)
	if err != nil {
		t.Fatalf("failed to marshal ingredient: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"conversions"`) {
		t.Errorf("expected 'conversions' key in JSON: %s", bodyStr)
	}
}

func TestGenerateIngredientsWithConversionsViewModel_NoConversions(t *testing.T) {
	ingID := uuid.New()
	ingredients := []database.GetIngredientListRow{
		{IngredientID: ingID, Name: "Water", Quantity: 1, Unit: "Cup"},
	}
	conversions := map[uuid.UUID][]database.Conversion{}

	result := GenerateIngredientsWithConversionsViewModel(ingredients, conversions)

	if len(result) != 1 {
		t.Fatalf("expected 1 ingredient, got %d", len(result))
	}

	if len(result[0].Conversions) != 0 {
		t.Errorf("expected 0 conversions, got %d", len(result[0].Conversions))
	}

	// conversions key should be omitted when empty
	body, err := json.Marshal(result[0])
	if err != nil {
		t.Fatalf("failed to marshal ingredient: %v", err)
	}
	if strings.Contains(string(body), `"conversions"`) {
		t.Errorf("expected 'conversions' to be omitted from JSON when empty: %s", string(body))
	}
}


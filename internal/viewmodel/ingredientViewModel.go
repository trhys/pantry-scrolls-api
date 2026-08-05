package viewmodel

import (
	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

type Conversion struct {
	FromUnit string  `json:"from_unit"`
	ToUnit   string  `json:"to_unit"`
	Ratio    float32 `json:"ratio"`
}

type Ingredient struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Quantity    float32      `json:"quantity,omitempty"`
	Unit        string       `json:"unit,omitempty"`
	Conversions []Conversion `json:"conversions,omitempty"`
}

type Unit struct {
	Name string `json:"name"`
}

type UnitsViewModel struct {
	Units []Unit `json:"units"`
}

func GenerateUnitsViewModel(conversions []database.Conversion) UnitsViewModel {
	model := UnitsViewModel{}

	for _, unit := range conversions {
		model.Units = append(model.Units, Unit{
			Name: unit.FromUnit,
		})
	}

	return model
}

func GenerateIngredientsViewModel(ingredients []database.GetIngredientListRow) []Ingredient {
	model := make([]Ingredient, 0, len(ingredients))
	for _, ing := range ingredients {
		model = append(model, Ingredient{
			ID:       ing.IngredientID,
			Name:     ing.Name,
			Quantity: ing.Quantity,
			Unit:     ing.Unit,
		})
	}

	return model
}

func GenerateIngredientsWithConversionsViewModel(ingredients []database.GetIngredientListRow, conversions map[uuid.UUID][]database.Conversion) []Ingredient {
	model := make([]Ingredient, 0, len(ingredients))
	for _, ing := range ingredients {
		convs := make([]Conversion, 0)
		for _, c := range conversions[ing.IngredientID] {
			convs = append(convs, Conversion{
				FromUnit: c.FromUnit,
				ToUnit:   c.ToUnit,
				Ratio:    c.Ratio,
			})
		}
		model = append(model, Ingredient{
			ID:          ing.IngredientID,
			Name:        ing.Name,
			Quantity:    ing.Quantity,
			Unit:        ing.Unit,
			Conversions: convs,
		})
	}

	return model
}

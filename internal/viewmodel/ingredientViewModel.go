package viewmodel

import (
	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

type Ingredient struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Quantity float32   `json:"quantity,omitempty"`
	Unit     string    `json:"unit,omitempty"`
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

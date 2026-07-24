package viewmodel

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// Consolidated separate shapes into one generic shape that can
// omit unneeded fields, and return one viewmodel of []Recipe
// for any endpoint

// We take optional fields as pointers to make them nil-able for better
// JSON marshalling when it goes to respond handlers

type Recipe struct {
	ID           uuid.UUID    `json:"id"`
	Title        string       `json:"title"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    *time.Time   `json:"updated_at"`
	UserID       *uuid.UUID   `json:"user_id"`
	Author       *string      `json:"author"`
	Description  *string      `json:"description"`
	ImageKey     *string      `json:"image_url"`
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions *string      `json:"instructions"`
	Quantity     *int32       `json:"quantity"`
	Likes        *int64       `json:"likes"`
	Liked        *bool        `json:"liked,omitempty"`
}

type RecipeViewModel struct {
	Recipes []Recipe `json:"recipes"`
}

func (builder *VMFactory) GenerateRecipeViewModel(data any, ingredients []Ingredient) RecipeViewModel {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var targetRecipes []Recipe

	switch val.Kind() {
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			recipe, err := builder.parseRecipe(item, ingredients)
			if err != nil {
				return RecipeViewModel{}
			}
			targetRecipes = append(targetRecipes, recipe)
		}

	case reflect.Struct:
		recipe, err := builder.parseRecipe(val, ingredients)
		if err != nil {
			return RecipeViewModel{}
		}
		targetRecipes = append(targetRecipes, recipe)

	default:
		return RecipeViewModel{}
	}

	return RecipeViewModel{Recipes: targetRecipes}
}

func (builder *VMFactory) parseRecipe(src reflect.Value, ingredients []Ingredient) (Recipe, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	if src.Kind() != reflect.Struct {
		return Recipe{}, fmt.Errorf("cannot parse non-struct type: %s", src.Kind())
	}

	var dest Recipe

	if f := src.FieldByName("ID"); f.IsValid() {
		dest.ID = f.Interface().(uuid.UUID)
	}
	if f := src.FieldByName("Title"); f.IsValid() {
		dest.Title = f.String()
	}
	if f := src.FieldByName("CreatedAt"); f.IsValid() {
		dest.CreatedAt = f.Interface().(time.Time)
	}
	if f := src.FieldByName("UpdatedAt"); f.IsValid() {
		t := f.Interface().(time.Time)
		if !t.IsZero() {
			dest.UpdatedAt = &t
		}
	}
	if f := src.FieldByName("UserID"); f.IsValid() {
		u := f.Interface().(uuid.UUID)
		dest.UserID = &u
	}
	if f := src.FieldByName("Author"); f.IsValid() {
		s := f.String()
		dest.Author = &s
	}
	if f := src.FieldByName("Description"); f.IsValid() {
		s := f.String()
		dest.Description = &s
	}
	if f := src.FieldByName("ImageKey"); f.IsValid() {
		s := fmt.Sprintf("%s/%s", builder.S3cdn, f.String())
		dest.ImageKey = &s
	}
	if f := src.FieldByName("Instructions"); f.IsValid() {
		s := f.String()
		dest.Instructions = &s
	}
	if f := src.FieldByName("Quantity"); f.IsValid() {
		q := int32(f.Int())
		dest.Quantity = &q
	}
	if f := src.FieldByName("Likes"); f.IsValid() {
		l := int64(f.Int())
		dest.Likes = &l
	}
	if f := src.FieldByName("Liked"); f.IsValid() {
		b := f.Bool()
		dest.Liked = &b
	}

	dest.Ingredients = ingredients

	return dest, nil
}

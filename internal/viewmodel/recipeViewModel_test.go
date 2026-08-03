package viewmodel

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateRecipeViewModelLikedField(t *testing.T) {
	builder := VMFactory{S3cdn: "https://cdn.example.com"}
	now := time.Now().UTC()

	t.Run("includes liked when source exposes field", func(t *testing.T) {
		model := builder.GenerateRecipeViewModel(struct {
			ID           uuid.UUID
			Title        string
			CreatedAt    time.Time
			UpdatedAt    time.Time
			UserID       uuid.UUID
			Author       string
			Description  string
			ImageKey     string
			Instructions string
			Likes        int64
			Liked        bool
		}{
			ID:           uuid.New(),
			Title:        "Liked Recipe",
			CreatedAt:    now,
			UpdatedAt:    now,
			UserID:       uuid.New(),
			Author:       "author",
			Description:  "desc",
			ImageKey:     "image-key",
			Instructions: "cook",
			Likes:        4,
			Liked:        true,
		}, nil)

		if len(model.Recipes) != 1 {
			t.Fatalf("expected 1 recipe, got %d", len(model.Recipes))
		}
		if model.Recipes[0].Liked == nil || !*model.Recipes[0].Liked {
			t.Fatalf("expected liked=true in view model")
		}
	})

	t.Run("omits liked when source does not expose field", func(t *testing.T) {
		model := builder.GenerateRecipeViewModel(struct {
			ID           uuid.UUID
			Title        string
			CreatedAt    time.Time
			UpdatedAt    time.Time
			UserID       uuid.UUID
			Author       string
			Description  string
			ImageKey     string
			Instructions string
			Likes        int64
		}{
			ID:           uuid.New(),
			Title:        "Anonymous Recipe",
			CreatedAt:    now,
			UpdatedAt:    now,
			UserID:       uuid.New(),
			Author:       "author",
			Description:  "desc",
			ImageKey:     "image-key",
			Instructions: "cook",
			Likes:        4,
		}, nil)

		body, err := json.Marshal(model)
		if err != nil {
			t.Fatalf("failed to marshal model: %v", err)
		}
		if model.Recipes[0].Liked != nil {
			t.Fatalf("expected liked to be nil when source field absent")
		}
		if strings.Contains(string(body), `"liked"`) {
			t.Fatalf("expected liked to be omitted from json: %s", string(body))
		}
	})

	t.Run("includes tags when source exposes []string field", func(t *testing.T) {
		model := builder.GenerateRecipeViewModel(struct {
			ID           uuid.UUID
			Title        string
			CreatedAt    time.Time
			UpdatedAt    time.Time
			UserID       uuid.UUID
			Author       string
			Description  string
			ImageKey     string
			Instructions string
			Likes        int64
			Tags         []string
		}{
			ID:           uuid.New(),
			Title:        "Tagged Recipe",
			CreatedAt:    now,
			UpdatedAt:    now,
			UserID:       uuid.New(),
			Author:       "author",
			Description:  "desc",
			ImageKey:     "image-key",
			Instructions: "cook",
			Likes:        4,
			Tags:         []string{"dinner", "easy"},
		}, nil)

		if len(model.Recipes) != 1 {
			t.Fatalf("expected 1 recipe, got %d", len(model.Recipes))
		}
		if got := model.Recipes[0].Tags; len(got) != 2 || got[0] != "dinner" || got[1] != "easy" {
			t.Fatalf("expected tags [dinner easy], got %v", got)
		}
	})

	t.Run("parses pq text array payload from sqlc interface field", func(t *testing.T) {
		model := builder.GenerateRecipeViewModel(struct {
			ID           uuid.UUID
			Title        string
			CreatedAt    time.Time
			UpdatedAt    time.Time
			UserID       uuid.UUID
			Author       string
			Description  string
			ImageKey     string
			Instructions string
			Likes        int64
			Tags         interface{}
		}{
			ID:           uuid.New(),
			Title:        "Tagged Recipe",
			CreatedAt:    now,
			UpdatedAt:    now,
			UserID:       uuid.New(),
			Author:       "author",
			Description:  "desc",
			ImageKey:     "image-key",
			Instructions: "cook",
			Likes:        4,
			Tags:         []byte("{dinner,easy}"),
		}, nil)

		if len(model.Recipes) != 1 {
			t.Fatalf("expected 1 recipe, got %d", len(model.Recipes))
		}
		if got := model.Recipes[0].Tags; len(got) != 2 || got[0] != "dinner" || got[1] != "easy" {
			t.Fatalf("expected tags [dinner easy], got %v", got)
		}
	})
}

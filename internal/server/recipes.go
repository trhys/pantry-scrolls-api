package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"mime"
	"net/http"
    "strings"

	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func (cfg *ApiConfig) handlerCreateRecipe(w http.ResponseWriter, r *http.Request) {
	// Request
	r.Body = http.MaxBytesReader(w, r.Body, 10 << 20)
	var req struct{
		Title 		string `json:"title"`
		Description	string `json:"description"`
		Ingredients 	[]struct{
			ID		uuid.UUID `json:"id"`
			Quantity 	float32 `json:"quantity"`
			Unit		string `json:"unit"`
		} `json:"ingredients"`
		Instructions 	string `json:"instructions"`

	}

	// Get request payload 
	jsonString := r.FormValue("payload")

	// Unmarshal JSON
	if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
		respondFail(w, 400, "Bad request", fmt.Errorf("Failed to unmarshal request body: %v", err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Request is valid - begin processing image file
	file, fileHeader, err := r.FormFile("image")
	key := cfg.ImagePlaceholder
	if err == nil {
		defer file.Close()
	
		mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
		if err != nil {
			respondFail(w, 401, "Couldn't parse media type", fmt.Errorf("Bad mime type in formfile: %v", err))
			return
		}

		if mediaType != "image/jpeg" && mediaType != "image/png" {
			respondFail(w, 401, "Invalid media type", fmt.Errorf("Must be jpg or png. Got: %s", mediaType))
			return
		}

		tmp, err := os.CreateTemp("", "image_upload")
		if err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("IO failure during image upload: %v", err))
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, fail := io.Copy(tmp, file)
		if fail != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("IO failure during image upload: %v", err))
			return
		}

		tmp.Seek(0, io.SeekStart)

		// Upload to s3
		key = uuid.New().String()
		if _, err := cfg.S3client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket: &cfg.S3bucket,
			Key: &key,
			Body: tmp,
			ContentType: &mediaType,
		}); err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed S3 put: %v", err))
			return
		}

	} else if err != nil {
		if err != http.ErrMissingFile {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed image upload: %v", err))
			return
		}
	}

	// Get username
	username, err := cfg.DB.GetName(r.Context(), requesterID)
	if err != nil {
		respondFail(w, 404, "Invalid user id", fmt.Errorf("Failed to get user from database: %v", err))
		return
	}

	// Query database
	query := database.CreateRecipeParams{
		Title: req.Title,
		Author: username,
		UserID: requesterID,
		Description: req.Description,
		ImageKey: key,
		Instructions: req.Instructions,
	}

	rec, err := cfg.DB.CreateRecipe(r.Context(), query)
	if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to create recipe: %v", err))
		return
	}

	// Connect all ingredients
	for _, ing := range req.Ingredients {
		query := database.AddToRecipeParams{
			RecipeID: rec.ID,
			IngredientID: ing.ID,
			Quantity: ing.Quantity,
			Unit: ing.Unit,
		}

		_, err := cfg.DB.AddToRecipe(r.Context(), query)
		if err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to add ingredient to recipe: %v", err))
			return
		}
	}

	i, err := cfg.DB.GetIngredientList(r.Context(), rec.ID)
	if err != nil {
		respondFail(w, 404, "Couldn't find ingredients", fmt.Errorf("Failed to find ingredients for recipe id: %s, ERROR: %v", rec.ID, err))
		return
	}

	respondJSON(w, 200, cfg.Vmf.GenerateRecipeFullViewModel(rec, viewmodel.GenerateIngredientsViewModel(i)))
}

// Get recipe by ID
func (cfg *ApiConfig) handlerGetRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(w, 404, "Invalid recipe id", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	rec, err := cfg.DB.GetRecipe(r.Context(), recipe_id)
	if err != nil {
		respondFail(w, 404, "Couldn't find recipe id", fmt.Errorf("Failed to find recipe with ID: %s, ERROR: %v", requested, err))
		return
	}

	i, err := cfg.DB.GetIngredientList(r.Context(), recipe_id)
	if err != nil {
		respondFail(w, 404, "Couldn't find ingredients", fmt.Errorf("Failed to find ingredients for recipe id: %s, ERROR: %v", requested, err))
		return
	}

	model := cfg.Vmf.GenerateRecipeFullViewModel(rec, viewmodel.GenerateIngredientsViewModel(i))

	respondJSON(w, 200, model)
}

// Get ten most recent recipes
func (cfg *ApiConfig) handlerGetRecipeList(w http.ResponseWriter, r *http.Request) {
	recipes, err := cfg.DB.GetRecipeList(r.Context())
	if err != nil {
		respondFail(w, 404, "Failed to retrieve recipe list", fmt.Errorf("Failed to get recipe list: %v", err))
		return
	}

	respondJSON(w, 200, cfg.Vmf.GenerateRecipeCardViewModel(recipes))
}

// Update recipe
func (cfg *ApiConfig) handlerUpdateRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(w, 404, "Invalid recipe id", fmt.Errorf("Failed to parse UUID %s : ERROR: %v", requested, err))
		return
	}

	// Request
	r.Body = http.MaxBytesReader(w, r.Body, 10 << 20)
	var req struct{
		Title 		string `json:"title"`
		Description	string `json:"description"`
		Ingredients 	[]struct{
			ID		uuid.UUID `json:"id"`
			Quantity 	float32 `json:"quantity"`
			Unit		string `json:"unit"`
		} `json:"ingredients"`
		Instructions 	string `json:"instructions"`

	}

	// Get request payload 
	jsonString := r.FormValue("payload")

	// Unmarshal JSON
	if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
		respondFail(w, 404, "Bad request", fmt.Errorf("Failed to unmarshal request body: %v", err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Verify ownership
	owner, err := cfg.DB.GetRecipeOwner(r.Context(), recipe_id)
	if err != nil || owner != requesterID {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get existing key
	key, err := cfg.DB.GetRecipeImageKey(r.Context(), recipe_id)
	if err != nil {
		log.Printf("Failed to get image key for recipe PUT: id - %s error: %v", recipe_id, err)
		key = uuid.New().String()
	}

	// Begin processing image file
	file, fileHeader, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
		if err != nil {
			respondFail(w, 401, "Couldn't parse media type", fmt.Errorf("Bad mime type in formfile: %v", err))
			return
		}

		if mediaType != "image/jpeg" && mediaType != "image/png" {
			respondFail(w, 401, "Invalid media type", fmt.Errorf("Must be jpg or png. Got: %s", mediaType))
			return
		}

		tmp, err := os.CreateTemp("", "image_upload")
		if err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("IO error during image upload: %v", err))
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, fail := io.Copy(tmp, file)
		if fail != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("IO error during image upload: %v", err))
			return
		}

		tmp.Seek(0, io.SeekStart)

		// Upload to s3
		if key == cfg.ImagePlaceholder {
			key = uuid.New().String()
		}

		if _, err := cfg.S3client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket: &cfg.S3bucket,
			Key: &key,
			Body: tmp,
			ContentType: &mediaType,
		}); err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("S3 Put object error: %v", err))
			return
		}

	} else if err != nil {
		if err != http.ErrMissingFile {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Error during S3 image upload: %v", err))
			return
		}
	}

	// Query database
	query := database.UpdateRecipeParams{
		Title: req.Title,
		Description: req.Description,
		ImageKey: key,
		Instructions: req.Instructions,
		ID: recipe_id,
	}

	rec, err := cfg.DB.UpdateRecipe(r.Context(), query)
	if err != nil {
		respondFail(w, 500, "Couldn't update recipe", fmt.Errorf("Database error: %v", err))
		return
	}

	// Update ingredients
	if err := cfg.DB.ClearFromRecipe(r.Context(), recipe_id); err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Database error: %v", err))
		return
	}

	for _, ing := range req.Ingredients {
		query := database.AddToRecipeParams{
			RecipeID: rec.ID,
			IngredientID: ing.ID,
			Quantity: ing.Quantity,
			Unit: ing.Unit,
		}

		_, err := cfg.DB.AddToRecipe(r.Context(), query)
		if err != nil {
			respondFail(w, 500, "Failed to add ingredient", fmt.Errorf("Couldn't perform AddToRecipe query: %v", err))
			return
		}
	}

	respondJSON(w, 204, nil)
}

// Delete recipe
func (cfg *ApiConfig) handlerDeleteRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(w, 404, "Invalid recipe id", fmt.Errorf("Couldn't parse UUID: %s ERROR: %v", requested, err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Verify ownership
	owner, err := cfg.DB.GetRecipeOwner(r.Context(), recipe_id)
	if err != nil || owner != requesterID {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Valid request - delete from database
	if err := cfg.DB.DeleteRecipe(r.Context(), recipe_id); err != nil {
		respondFail(w, 404, "Couldn't delete recipe", fmt.Errorf("Failed to delete recipe: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

func (cfg *ApiConfig) handlerExploreFeed(w http.ResponseWriter, r *http.Request) {
  query := r.URL.Query().Get("search")
  if query != "" {
    feed, err := cfg.DB.GetRecipesFromQuery(r.Context(), strings.ToLower(query))
    if err != nil {
      respondFail(w, 404, "No recipes matched the query params", fmt.Errorf("recipes query error: %v", err))
      return
    }

    respondJSON(w, 200, cfg.Vmf.GenerateRecipeCardViewModel(feed))
  }
}

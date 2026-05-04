package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"html/template"
	"mime"
	"net/http"

	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func (cfg *apiConfig) handlerCreateRecipe(w http.ResponseWriter, r *http.Request) {
	// Request
	r.Body = http.MaxBytesReader(w, r.Body, 10 << 20)
	var req struct{
		Title 		string `json:"title"`
		UserID 		uuid.UUID `json:"user_id"`
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
		respondFail(w, 500, "Failed to unmarshal payload", err)
		return
	}

	// Get username
	username, err := cfg.db.GetName(r.Context(), req.UserID)
	if err != nil {
		respondFail(w, 404, "Invalid user id", err)
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	if requesterID != req.UserID {
                respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
                return
        }

	// Request is valid - begin processing image file
	file, fileHeader, err := r.FormFile("image")
	key := cfg.imagePlaceholder
	if err == nil {
		defer file.Close()
	
		mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
		if err != nil {
			respondFail(w, 401, "Couldn't parse media type", err)
			return
		}

		if mediaType != "image/jpeg" && mediaType != "image/png" {
			respondFail(w, 401, "Invalid media type", fmt.Errorf("Must be jpg or png. Got: %s", mediaType))
			return
		}

		tmp, err := os.CreateTemp("", "image_upload")
		if err != nil {
			respondFail(w, 500, "Something went wrong", err)
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, fail := io.Copy(tmp, file)
		if fail != nil {
			respondFail(w, 500, "Couldn't save image", err)
			return
		}

		tmp.Seek(0, io.SeekStart)

		// Upload to s3
		key = uuid.New().String()
		if _, err := cfg.s3client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket: &cfg.s3bucket,
			Key: &key,
			Body: tmp,
			ContentType: &mediaType,
		}); err != nil {
			respondFail(w, 500, "Failed to upload to s3 bucket", err)
			return
		}

	} else if err != nil {
		if err != http.ErrMissingFile {
			respondFail(w, 500, "Something went wrong during upload", err)
			return
		}
	}

	// Query database
	query := database.CreateRecipeParams{
		Title: req.Title,
		Author: username,
		UserID: req.UserID,
		Description: req.Description,
		ImageKey: key,
		Instructions: req.Instructions,
	}

	rec, err := cfg.db.CreateRecipe(r.Context(), query)
	if err != nil {
		respondFail(w, 404, "Couldn't create recipe", err)
		return
	}

	// Connect all ingredients
	ingredients := []viewmodel.Ingredient{}
	for _, ing := range req.Ingredients {
		query := database.AddToRecipeParams{
			RecipeID: rec.ID,
			IngredientID: ing.ID,
			Quantity: ing.Quantity,
			Unit: ing.Unit,
		}

		_, err := cfg.db.AddToRecipe(r.Context(), query)
		if err != nil {
			respondFail(w, 500, "Failed to add ingredient", err)
			return
		}

		ingName, err := cfg.db.GetIngredientName(r.Context(), ing.ID)
		if err != nil {
			respondFail(w, 500, "Couldn't fetch ingredient name", err)
			return
		}

		ingredients = append(ingredients, viewmodel.Ingredient{
			ID: ing.ID,
			Name: ingName,
			Quantity: ing.Quantity,
			Unit: ing.Unit,
		})
	}

	respondJSON(w, 201, cfg.vmf.GenerateRecipeFullViewModel(rec, ingredients))
}

// Get recipe by ID
func (cfg *apiConfig) handlerGetRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(w, 404, "Invalid recipe id", err)
		return
	}

	rec, err := cfg.db.GetRecipe(r.Context(), recipe_id)
	if err != nil {
		respondFail(w, 404, "Couldn't find recipe id", err)
		return
	}

	i, err := cfg.db.GetIngredientList(r.Context(), recipe_id)
	if err != nil {
		respondFail(w, 404, "Couldn't find ingredients", err)
		return
	}

	ingredients := []viewmodel.Ingredient{}
	for _, ing := range i {
		ingredients = append(ingredients, viewmodel.Ingredient{
			ID: ing.IngredientID,
			Name: ing.Name,
			Quantity: ing.Quantity,
			Unit: ing.Unit,
		})
	}

	model := cfg.vmf.GenerateRecipeFullViewModel(rec, ingredients)

	if r.Header.Get("Accept") == "application/json" {
		respondJSON(w, 200, model)
		return
	}

	tmpl, _ := template.ParseFiles(filepath.Join("app", "templates", "recipe-viewer.html"))
	tmpl.Execute(w, model)
}

// Get ten most recent recipes
func (cfg *apiConfig) handlerGetRecipeList(w http.ResponseWriter, r *http.Request) {
	recipes, err := cfg.db.GetRecipeList(r.Context())
	if err != nil {
		respondFail(w, 404, "Failed to retrieve recipe list", err)
		return
	}

	respondJSON(w, 200, cfg.vmf.GenerateRecipeCardViewModel(recipes))
}

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

// NOTE: Probably will be removing creation here in some capacity in the future. Right now its useless, if i have ingredients to add im more likely to go ahead and write a migration with a large number of them. May allow users to create their own but for now it breaks the intention of integrating shopping Apis.

type ingredient struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	ImageKey  string    `json:"image_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (cfg *ApiConfig) handlerCreateIngredient(w http.ResponseWriter, r *http.Request) {
	// Authorization
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondFail(r, w, 401, "Failed to retrieve bearer token", err)
		return
	}

	subject, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		respondFail(r, w, 401, "Couldn't validate token", err)
		return
	}

	admin, err := cfg.DB.CheckAdmin(r.Context(), subject)
	if err != nil {
		respondFail(r, w, 500, "Something went wrong", err)
		return
	} else if admin == false {
		respondFail(r, w, 403, "Unauthorized access", fmt.Errorf("Must be administrator"))
		return
	}

	// User is admin --- proceed

	var req struct {
		Name string `json:"name"`
	}

	// Get request payload
	jsonString := r.FormValue("payload")

	// Unmarshal JSON
	if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
		respondFail(r, w, 500, "Failed to unmarshal payload", err)
		return
	}

	file, fileHeader, err := r.FormFile("image")
	key := uuid.New().String()
	if err == nil {
		defer file.Close()

		mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
		if err != nil {
			respondFail(r, w, 400, "Couldn't parse media type", err)
			return
		}

		if mediaType != "image/jpeg" && mediaType != "image/png" {
			respondFail(r, w, 400, "Invalid media type", fmt.Errorf("Must be jpg or png. Got: %s", mediaType))
			return
		}

		tmp, err := os.CreateTemp("", "image_upload")
		if err != nil {
			respondFail(r, w, 500, "Something went wrong", err)
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, fail := io.Copy(tmp, file)
		if fail != nil {
			respondFail(r, w, 500, "Couldn't save image", err)
			return
		}

		tmp.Seek(0, io.SeekStart)

		// Upload to s3
		if _, err := cfg.S3client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:      &cfg.S3bucket,
			Key:         &key,
			Body:        tmp,
			ContentType: &mediaType,
		}); err != nil {
			respondFail(r, w, 500, "Failed to upload to s3 bucket", err)
			return
		}

	} else if err != nil {
		if err != http.ErrMissingFile {
			respondFail(r, w, 400, "Something went wrong during upload", err)
			return
		}
	}

	query := database.CreateIngredientParams{
		Name:     req.Name,
		ImageKey: key,
	}

	ing, err := cfg.DB.CreateIngredient(r.Context(), query)
	if err != nil {
		respondFail(r, w, 500, "Database error during write", err)
		return
	}

	res := ingredient{
		ID:        ing.ID,
		Name:      ing.Name,
		ImageKey:  ing.ImageKey,
		CreatedAt: ing.CreatedAt,
		UpdatedAt: ing.UpdatedAt,
	}

	respondJSON(w, 200, res)
}

// Grabs the full collection of ingredients
func (cfg *ApiConfig) handlerGetIngredientBase(w http.ResponseWriter, r *http.Request) {
	ingredients, err := cfg.DB.GetIngredients(r.Context())
	if err != nil {
		respondFail(r, w, 404, "Failed to retrieve ingredients from database", err)
		return
	}

	type resp struct {
		Ingredients []ingredient `json:"ingredients"`
	}

	res := resp{}
	for _, i := range ingredients {
		res.Ingredients = append(res.Ingredients, ingredient{
			ID:   i.ID,
			Name: i.Name,
		})
	}

	respondJSON(w, 200, res)
}

// Gets collection of units for ingredient by id
func (cfg *ApiConfig) handlerGetUnits(w http.ResponseWriter, r *http.Request) {
	if !r.URL.Query.Has("id") {
		respondFail(r, w, 404, "invalid ingredient id", fmt.Errorf("Empty id in units query"))
		return
	}
	
	val := r.URL.Query.Get("id")
	id, err := uuid.Parse(val)
	if err != nil {
		respondFail(r, w, 404, "Invalid ingredient id", fmt.Errorf("Failed to get id from path: %v", err))
		return
	}

	// make sure ingredient exists
	if _, err := cfg.DB.GetIngredientName(r.Context(), id); err != nil {
		respondFail(r, w, 404, "Invalid ingredient id", fmt.Errorf("Can't find units for nonexistent ingredient: %v", err))
		return
	}

	conversions, err := cfg.DB.GetConversionsByID(r.Context(), id)
	if err != nil {
		respondFail(r, w, 404, "Couldn't find units", fmt.Errorf("Failed to get units for ingredient: %v", err))
		return
	}

	respondJSON(w, 200, viewmodel.GenerateUnitsViewModel(conversions))
}

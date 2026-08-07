package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	util "github.com/trhys/Recipe-Repo-2/internal/utility"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

// helper to verify ingredients have a non nill unit and quantity set
func verifyIngredient(ings []struct) error {
	for _, i := range ings {
		if i.Quantity == 0 | i.Units == "" {
			return fmt.Errorf("Unset quantity or unit in ingredients")
		}
	}
	return nil
}
	
func (cfg *ApiConfig) handlerCreateRecipe(w http.ResponseWriter, r *http.Request) {
	// Request
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Ingredients []struct {
			ID       uuid.UUID `json:"id"`
			Quantity float32   `json:"quantity"`
			Unit     string    `json:"unit"`
		} `json:"ingredients"`
		Instructions string `json:"instructions"`
		Tags		 []string `json:"tags"`
	}

	// Get request payload
	jsonString := r.FormValue("payload")

	// Unmarshal JSON
	if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Failed to unmarshal request body: %v", err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// validate ingredients is non empty && units/quantity is set
	if err := util.CheckNilSlice([][]any{req.Ingredients}); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Request unmarshalled with empty ingredients - %v", err))
		return
	}

	if err := verifyIngredients(req.Ingredients); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Request unmarshalled with unset ingredient vars - %v", err))
		return
	}

	// Request is valid - begin processing image file
	file, fileHeader, err := r.FormFile("image")
	key := cfg.ImagePlaceholder
	if err == nil {
		defer file.Close()

		mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
		if err != nil {
			respondFail(r, w, 401, "Couldn't parse media type", fmt.Errorf("Bad mime type in formfile: %v", err))
			return
		}

		if mediaType != "image/jpeg" && mediaType != "image/png" {
			respondFail(r, w, 401, "Invalid media type", fmt.Errorf("Must be jpg or png. Got: %s", mediaType))
			return
		}

		tmp, err := os.CreateTemp("", "image_upload")
		if err != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("IO failure during image upload: %v", err))
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, fail := io.Copy(tmp, file)
		if fail != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("IO failure during image upload: %v", err))
			return
		}

		tmp.Seek(0, io.SeekStart)

		// Upload to s3
		key = uuid.New().String()
		if _, err := cfg.S3client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:      &cfg.S3bucket,
			Key:         &key,
			Body:        tmp,
			ContentType: &mediaType,
		}); err != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Failed S3 put: %v", err))
			return
		}

	} else if err != nil {
		if err != http.ErrMissingFile {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Failed image upload: %v", err))
			return
		}
	}

	// Get username
	username, err := cfg.DB.GetName(r.Context(), requesterID)
	if err != nil {
		respondFail(r, w, 404, "Invalid user id", fmt.Errorf("Failed to get user from database: %v", err))
		return
	}

	// Begin write tx
	err = cfg.withTx(r.Context(), func(qtx *database.Queries) error {
		query := database.CreateRecipeParams{
			Title:        req.Title,
			Author:       username,
			UserID:       requesterID,
			Description:  req.Description,
			ImageKey:     key,
			Instructions: req.Instructions,
		}
	
		recipeID, err := qtx.CreateRecipe(r.Context(), query)
		if err != nil {
			return err
		}
	
		// Connect all ingredients
		for _, ing := range req.Ingredients {
			query := database.AddToRecipeParams{
				RecipeID:     recipeID,
				IngredientID: ing.ID,
				Quantity:     ing.Quantity,
				Unit:         ing.Unit,
			}
	
			_, err := qtx.AddToRecipe(r.Context(), query)
			if err != nil {
				return err
			}
		}
	
		queryTags := database.AddRecipeTagsParams{
			RecipeID:	recipeID,
			Tags:		req.Tags,
		}
	
		if err := qtx.AddRecipeTags(r.Context(), queryTags); err != nil {
			return err
		}
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23503":
				respondFail(r, w, 400, "Bad request", fmt.Errorf("handlerCreateRecipe tx error --- invalid foreign key: %v", err))
				return
			case "23505":
				respondFail(r, w, 400, "Bad request", fmt.Errorf("handlerCreateRecipe tx error --- duplicate value: %v", err))
				return
			}
		}
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("handlerCreateRecipe transaction failed: %v", err))
		return
	}

	type resp struct {
		ID uuid.UUID `json:"id"`
	}

	respondJSON(w, 201, resp{ID: recipeID})
}

// Get recipe by ID with ingredient conversions for editor hydration
func (cfg *ApiConfig) handlerGetRecipeEdit(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	requesterID, _ := requesterUserID(r)

	owner, err := cfg.DB.GetRecipeOwner(r.Context(), recipe_id)
	if err != nil || owner != requesterID {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %v", requesterID))
		return
	}

	rec, err := cfg.DB.GetAuthedRecipe(r.Context(), database.GetAuthedRecipeParams{
			ID:     recipe_id,
			UserID: requesterID,
	})
	if err != nil {
		respondFail(r, w, 404, "Couldn't find recipe id", fmt.Errorf("Failed to find recipe with ID: %s, ERROR: %v", requested, err))
		return
	}

	ingredientList, err := cfg.DB.GetIngredientList(r.Context(), recipe_id)
	if err != nil {
		respondFail(r, w, 404, "Couldn't find ingredients", fmt.Errorf("Failed to find ingredients for recipe id: %s, ERROR: %v", requested, err))
		return
	}

	conversionMap := make(map[uuid.UUID][]database.Conversion)
	for _, ing := range ingredientList {
		convs, err := cfg.DB.GetConversionsByID(r.Context(), ing.IngredientID)
		if err != nil {
			respondFail(r, w, 500, "Couldn't load conversions", fmt.Errorf("Failed to get conversions for ingredient %s: %v", ing.IngredientID, err))
			return
		}
		conversionMap[ing.IngredientID] = convs
	}

	ingredients := viewmodel.GenerateIngredientsWithConversionsViewModel(ingredientList, conversionMap)
	model := cfg.Vmf.GenerateRecipeViewModel(rec, ingredients)

	respondJSON(w, 200, model)
}

// Get recipe by ID
func (cfg *ApiConfig) handlerGetRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Failed to parse UUID: %v", err))
		return
	}

	requesterID, ok := requesterUserID(r)

	var rec any
	if ok {
		rec, err = cfg.DB.GetAuthedRecipe(r.Context(), database.GetAuthedRecipeParams{
			ID:     recipe_id,
			UserID: requesterID,
		})
	} else {
		rec, err = cfg.DB.GetRecipe(r.Context(), recipe_id)
	}
	if err != nil {
		respondFail(r, w, 404, "Couldn't find recipe id", fmt.Errorf("Failed to find recipe with ID: %s, ERROR: %v", requested, err))
		return
	}

	i, err := cfg.DB.GetIngredientList(r.Context(), recipe_id)
	if err != nil {
		respondFail(r, w, 404, "Couldn't find ingredients", fmt.Errorf("Failed to find ingredients for recipe id: %s, ERROR: %v", requested, err))
		return
	}

	model := cfg.Vmf.GenerateRecipeViewModel(rec, viewmodel.GenerateIngredientsViewModel(i))

	respondJSON(w, 200, model)
}

// Get ten most liked recipes or total number of recipes
func (cfg *ApiConfig) handlerGetRecipeList(w http.ResponseWriter, r *http.Request) {
	// return early with count if total query
	if r.URL.Query().Get("total") == "true" {
		total, err := cfg.DB.GetTotalRecipes(r.Context())
		if err != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Failed total recipes query: %v", err))
			return
		}
		respondJSON(w, 200, struct {
			Total int64 `json:"total"`
		}{Total: total})
		return
	}

	requesterID, ok := requesterUserID(r)

	var (
		recipes any
		err     error
	)
	if ok {
		recipes, err = cfg.DB.GetAuthedRecipeList(r.Context(), requesterID)
	} else {
		recipes, err = cfg.DB.GetRecipeList(r.Context())
	}
	if err != nil {
		respondFail(r, w, 404, "Failed to retrieve recipe list", fmt.Errorf("Failed to get recipe list: %v", err))
		return
	}

	respondJSON(w, 200, cfg.Vmf.GenerateRecipeViewModel(recipes, nil))
}

// Update recipe
func (cfg *ApiConfig) handlerUpdateRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Failed to parse UUID %s : ERROR: %v", requested, err))
		return
	}

	// Request
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Ingredients []struct {
			ID       uuid.UUID `json:"id"`
			Quantity float32   `json:"quantity"`
			Unit     string    `json:"unit"`
		} `json:"ingredients"`
		Instructions string `json:"instructions"`
		Tags		 []string `json:"tags"`
	}

	// Get request payload
	jsonString := r.FormValue("payload")

	// Unmarshal JSON
	if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
		respondFail(r, w, 404, "Bad request", fmt.Errorf("Failed to unmarshal request body: %v", err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Verify ownership
	owner, err := cfg.DB.GetRecipeOwner(r.Context(), recipe_id)
	if err != nil || owner != requesterID {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Get existing key
	key, err := cfg.DB.GetRecipeImageKey(r.Context(), recipe_id)
	if err != nil {
		slog.Warn("Failed to get image key", "recipe_id", recipe_id, "error", err)
		key = uuid.New().String()
	}

	// validate ingredients is non empty && units/quantity is set
	if err := util.CheckNilSlice([][]any{req.Ingredients}); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Request unmarshalled with empty ingredients - %v", err))
		return
	}

	if err := verifyIngredients(req.Ingredients); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Request unmarshalled with unset ingredient vars - %v", err))
		return
	}

	// Begin processing image file
	file, fileHeader, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
		if err != nil {
			respondFail(r, w, 401, "Couldn't parse media type", fmt.Errorf("Bad mime type in formfile: %v", err))
			return
		}

		if mediaType != "image/jpeg" && mediaType != "image/png" {
			respondFail(r, w, 401, "Invalid media type", fmt.Errorf("Must be jpg or png. Got: %s", mediaType))
			return
		}

		tmp, err := os.CreateTemp("", "image_upload")
		if err != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("IO error during image upload: %v", err))
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, fail := io.Copy(tmp, file)
		if fail != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("IO error during image upload: %v", err))
			return
		}

		tmp.Seek(0, io.SeekStart)

		// Upload to s3
		if key == cfg.ImagePlaceholder {
			key = uuid.New().String()
		}

		if _, err := cfg.S3client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:      &cfg.S3bucket,
			Key:         &key,
			Body:        tmp,
			ContentType: &mediaType,
		}); err != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("S3 Put object error: %v", err))
			return
		}

	} else if err != nil {
		if err != http.ErrMissingFile {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Error during S3 image upload: %v", err))
			return
		}
	}

	// Begin write tx
	err = cfg.withTx(r.Context(), func(qtx *database.Queries) error {
		query := database.UpdateRecipeParams{
			Title:        req.Title,
			Description:  req.Description,
			ImageKey:     key,
			Instructions: req.Instructions,
			ID:           recipe_id,
		}
	
		rec, err := qtx.UpdateRecipe(r.Context(), query)
		if err != nil {
			return err
		}
	
		// Update ingredients
		if err := qtx.ClearFromRecipe(r.Context(), recipe_id); err != nil {
			return err
		}
	
		for _, ing := range req.Ingredients {
			query := database.AddToRecipeParams{
				RecipeID:     rec.ID,
				IngredientID: ing.ID,
				Quantity:     ing.Quantity,
				Unit:         ing.Unit,
			}
	
			_, err := qtx.AddToRecipe(r.Context(), query)
			if err != nil {
				return err
			}
		}
	
		queryTags := database.AddRecipeTagsParams{
			RecipeID:	recipe_id,
			Tags:		req.Tags,
		}
	
		// clear existing tags first
		if err := qtx.ResetRecipeTags(r.Context(), recipe_id); err != nil {
			return err
		}
	
		if err := qtx.AddRecipeTags(r.Context(), queryTags); err != nil {
			return err
		}
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23503":
				respondFail(r, w, 400, "Bad request", fmt.Errorf("handlerUpdateRecipe tx invalid foreign key: %v", err))
				return
			case "23505":
				respondFail(r, w, 400, "Bad request", fmt.Errorf("handlerUpdateRecipe tx duplicate value: %v", err))
				return
			}
		}
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("handlerUpdateRecipe transaction failed: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

// Delete recipe
func (cfg *ApiConfig) handlerDeleteRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Couldn't parse UUID: %s ERROR: %v", requested, err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Verify ownership
	owner, err := cfg.DB.GetRecipeOwner(r.Context(), recipe_id)
	if err != nil || owner != requesterID {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// Valid request - delete from database
	err = cfg.withTx(r.Context(), func(qtx *database.Queries) error {
		if err := qtx.DeleteRecipe(r.Context(), recipe_id); err != nil {
			return err
		}
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23503":
				respondFail(r, w, 400, "Bad request", fmt.Errorf("handlerDeleteRecipe --- invalid foreign key: %v", err))
				return
			case "23505":
				respondFail(r, w, 400, "Bad request", fmt.Errorf("handlerDeleteRecipe --- duplicate value: %v", err))
				return
			}
		}
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("handlerDeleteRecipe transaction failed: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

func (cfg *ApiConfig) handlerExploreFeed(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	// find url query
	title, err := util.SanitizeSearchQuery(queryParams.Get("title"))
	if err != nil {
		respondFail(r, w, 400, "Bad request", err)
		return
	}
	author, err := util.SanitizeSearchQuery(queryParams.Get("author"))
	if err != nil {
		respondFail(r, w, 400, "Bad request", err)
		return
	}
	// tag, err := util.SanitizeSearchQuery(queryParams.Get("tag"))
	// if err != nil {
	// 	respondFail(r, w, 400, "Bad request", err)
	// 	return
	// }

	requesterID, ok := requesterUserID(r)

	var feed any
	if title != "" {
		if ok {
			feed, err = cfg.DB.GetAuthedRecipesFromTitleQuery(r.Context(), database.GetAuthedRecipesFromTitleQueryParams{
				Query:  title,
				UserID: requesterID,
			})
		} else {
			feed, err = cfg.DB.GetRecipesFromTitleQuery(r.Context(), title)
		}
		if err != nil {
			respondFail(r, w, 404, "No recipes matched the query params", fmt.Errorf("recipes query error: %v", err))
			return
		}
		respondJSON(w, 200, cfg.Vmf.GenerateRecipeViewModel(feed, nil))
        return
	} else if author != "" {
		if ok {
			feed, err = cfg.DB.GetAuthedRecipesFromAuthorQuery(r.Context(), database.GetAuthedRecipesFromAuthorQueryParams{
				Query:  author,
				UserID: requesterID,
			})
		} else {
			feed, err = cfg.DB.GetRecipesFromAuthorQuery(r.Context(), author)
		}
		if err != nil {
			respondFail(r, w, 404, "No recipes matched the query params", fmt.Errorf("recipes query error: %v", err))
			return
		}
		respondJSON(w, 200, cfg.Vmf.GenerateRecipeViewModel(feed, nil))
        return
	} else {
		if ok {
			feed, err = cfg.DB.GetAuthedRecipesFromNilQuery(r.Context(), requesterID)
		} else {
			feed, err = cfg.DB.GetRecipesFromNilQuery(r.Context())
		}
		if err != nil {
			respondFail(r, w, 404, "No recipes found", fmt.Errorf("recipes query error: %v", err))
			return
		}
		respondJSON(w, 200, cfg.Vmf.GenerateRecipeViewModel(feed, nil))
	}
}

func (cfg *ApiConfig) handlerLikeRecipe(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Couldn't parse UUID: %s ERROR: %v", requested, err))
		return
	}

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	// if there is no record of the like, add a like entry, otherwise delete it
	if err := cfg.DB.LikeRecipe(r.Context(), database.LikeRecipeParams{
		UserID:   requesterID,
		RecipeID: recipe_id,
	}); err != nil {
		if err.(*pq.Error).Code == "23505" {
			if err := cfg.DB.UnlikeRecipe(r.Context(), database.UnlikeRecipeParams{
				UserID:   requesterID,
				UserID_2: recipe_id,
			}); err != nil {
				respondFail(r, w, 500, "something went wrong", fmt.Errorf("Query Failure (UnlikeRecipe): %v", err))
				return
			}
		} else {
			respondFail(r, w, 500, "something went wrong", fmt.Errorf("Query Failure (LikeRecipe): %v", err))
			return
		}
	}

	respondJSON(w, 204, nil)
}

func (cfg *ApiConfig) handlerGetLikes(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Couldn't parse UUID: %s ERROR: %v", requested, err))
		return
	}

	likes, err := cfg.DB.GetLikes(r.Context(), recipe_id)
	if err != nil {
		respondFail(r, w, 404, "Couldn't get likes count", fmt.Errorf("Query Failure (GetLikes) for recipe id: %v ERROR: %v", recipe_id, err))
		return
	}

	resp := struct {
		Likes int64 `json:"likes"`
	}{
		Likes: likes,
	}

	respondJSON(w, 200, resp)
}

// returns true/false whether user has liked recipe @ id
func (cfg *ApiConfig) handlerCheckLiked(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("recipe_id")
	recipe_id, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid recipe id", fmt.Errorf("Couldn't parse UUID: %s ERROR: %v", requested, err))
		return
	}

	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

	liked, err := cfg.DB.CheckLiked(r.Context(), database.CheckLikedParams{UserID: requesterID, RecipeID: recipe_id})
	if err != nil {
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Query failed (CheckLiked): %v", err))
		return
	}

	resp := struct {
		Liked bool `json:"liked"`
	}{
		Liked: liked,
	}

	respondJSON(w, 200, resp)
}

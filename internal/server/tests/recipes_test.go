package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestCRUDRecipe(t *testing.T) {
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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	mockAwsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("mockadmin", "mockpassword", "")),
	)
	if err != nil {
		t.Fatalf("failed to build mock aws config: %v", err)
	}

	cfg.S3client = s3.NewFromConfig(mockAwsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
		o.UsePathStyle = true
	})

	testUser := struct {
		input []byte
	}{
		input: []byte(`{"email":"tim@test.com", "password": "password", "name": "tim"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("Failed to create user for test")
	}

	// Login test user
	login := []byte(`{"email":"tim@test.com", "password": "password"}`)
	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("Failed to login test user")
	}

	var user vm.SessionViewModel
	if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	// Get session cookies
	response := w.Result()
	cookies := response.Cookies()
	var jwt *http.Cookie
	var rt *http.Cookie
	for _, c := range cookies {
		if c.Name == "jwt" {
			jwt = c
		} else if c.Name == "refresh_token" {
			rt = c
		}
	}

	recipePayload := struct {
		Title string `json:"title"`
		Desc  string `json:"description"`
		Ing   []struct {
			ID       uuid.UUID `json:"id"`
			Quantity float32   `json:"quantity"`
			Unit     string    `json:"unit"`
		} `json:"ingredients"`
		Inst string `json:"instructions"`
	}{
		Title: "Classic Spaghetti Bolognese",
		Desc:  "A hearty Italian meat sauce slow-simmered with tomatoes and aromatics, served over al dente spaghetti.",
		Inst:  "1. Cook spaghetti... 2. Brown beef...",
	}

	testIngredients := []struct {
		Name     string
		Quantity float32
		Unit     string
	}{
		{"Spaghetti", 12, "Ounce"},
		{"Ground Beef", 1, "Pound"},
		{"Yellow Onion", 1, "Count"},
		{"Garlic", 4, "Count"},
		{"Carrots", 1, "Count"},
		{"Celery", 2, "Count"},
		{"Crushed Tomatoes", 1, "Cup"},
		{"Tomato Paste", 2, "Tablespoon"},
		{"Extra Virgin Olive Oil", 2, "Tablespoon"},
		{"Dried Oregano", 1, "Teaspoon"},
		{"Kosher Salt", 1, "Teaspoon"},
		{"Ground Black Pepper", 0.5, "Teaspoon"},
		{"Parmesan Cheese", 0.5, "Cup"},
	}

	for _, ing := range testIngredients {
		id, err := cfg.DB.GetIngredientFromName(context.Background(), ing.Name)
		if err != nil {
			t.Fatalf("Failed to resolve UUID for ingredient %s: %v", ing.Name, err)
		}

		recipePayload.Ing = append(recipePayload.Ing, struct {
			ID       uuid.UUID `json:"id"`
			Quantity float32   `json:"quantity"`
			Unit     string    `json:"unit"`
		}{
			ID:       id,
			Quantity: ing.Quantity,
			Unit:     ing.Unit,
		})
	}

	payloadJSON, err := json.Marshal(recipePayload)
	if err != nil {
		t.Fatalf("Failed to marshal recipe payload: %v", err)
	}

	// Write multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("payload", string(payloadJSON)); err != nil {
		t.Fatalf("failed to write payload field: %v", err)
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="Spaghetti.png"`)
	h.Set("Content-Type", "image/png")

	fileWriter, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create file part: %v", err)
	}

	mockImageBytes := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR...")
	if _, err := fileWriter.Write(mockImageBytes); err != nil {
		t.Fatalf("failed to write mock image bytes: %v", err)
	}

	writer.Close()

	var testRecipeID uuid.UUID

	// Create
	t.Run("create recipe", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/recipes", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), "userID", user.ID)
		req = req.WithContext(ctx)

		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Errorf("Failed to create recipe: got status %d", w.Code)
			return
		}

		responseResult := w.Result()
		defer responseResult.Body.Close()

		var responseBody struct {
			ID uuid.UUID `json:"id"`
		}

		decoder := json.NewDecoder(responseResult.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&responseBody); err != nil {
			t.Errorf("Response structural validation failed: %v", err)
		}

		// get id for next tests
		testRecipeID = responseBody.ID
	})

	// add a like to the recipe
	t.Run("like recipe", func(t *testing.T) {
		url := fmt.Sprintf("/api/recipes/%s/likes", testRecipeID.String())
		req := httptest.NewRequest("PUT", url, nil)

		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Failed to like recipe: got status %d", w.Code)
			return
		}
	})

	// Read
	t.Run("get recipe", func(t *testing.T) {
		// we're getting the 10 most liked recipes at this endpoint
		// highest likes should be index 0
		req := httptest.NewRequest("GET", "/api/recipes", nil)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to get recipes: got status %d", w.Code)
			return
		}

		responseResult := w.Result()
		defer responseResult.Body.Close()

		var responseBody vm.RecipeViewModel
		decoder := json.NewDecoder(responseResult.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&responseBody); err != nil {
			t.Errorf("Response structural validation failed: %v", err)
		}

		// this block assumes the old query returns - may change and uncomment

		// if responseBody.Recipes[0].Title != recipePayload.Title {
		// 	t.Errorf("Expected recipe title: %s got %s", recipePayload.Title, responseBody.Recipes[0].Title)
		// }
	})

	// Update
	t.Run("update recipe", func(t *testing.T) {
		url := "/api/recipes/" + testRecipeID.String()
		req := httptest.NewRequest("GET", url, nil)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to get recipe: got status %d", w.Code)
			return
		}

		responseResult := w.Result()
		defer responseResult.Body.Close()

		var responseBody vm.RecipeViewModel
		decoder := json.NewDecoder(responseResult.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&responseBody); err != nil {
			t.Errorf("Response structural validation failed: %v", err)
		}

		if responseBody.Recipes[0].ID != testRecipeID {
			t.Errorf("Expected recipe id: %v got %v", testRecipeID, responseBody.Recipes[0].ID)
		}

		// the update endpoint takes this shape (decoder will silently fail
		// if we give any unknown fields)
		// for simplicity we nil the ingredients slice and check for a new title

		requestBody := struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Ingredients []struct {
				ID       uuid.UUID `json:"id"`
				Quantity float32   `json:"quantity"`
				Unit     string    `json:"unit"`
			} `json:"ingredients"`
			Instructions string `json:"instructions"`
		}{
			Title:        "new title",
			Description:  *responseBody.Recipes[0].Description,
			Ingredients:  nil,
			Instructions: *responseBody.Recipes[0].Instructions,
		}

		data, _ := json.Marshal(requestBody)

		// write update body
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		if err := writer.WriteField("payload", string(data)); err != nil {
			t.Fatalf("failed to write payload field: %v", err)
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="image"; filename="Spaghetti.png"`)
		h.Set("Content-Type", "image/png")

		fileWriter, err := writer.CreatePart(h)
		if err != nil {
			t.Fatalf("failed to create file part: %v", err)
		}

		mockImageBytes := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR...")
		if _, err := fileWriter.Write(mockImageBytes); err != nil {
			t.Fatalf("failed to write mock image bytes: %v", err)
		}

		writer.Close()

		// send update
		req = httptest.NewRequest("PUT", url, body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Failed to update recipe: got status %d", w.Code)
			return
		}

		// get recipe again to verify changed title
		req = httptest.NewRequest("GET", url, nil)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to get recipe: got status %d", w.Code)
			return
		}

		updateResult := w.Result()
		defer updateResult.Body.Close()

		var updateBody vm.RecipeViewModel
		newDecoder := json.NewDecoder(updateResult.Body)
		newDecoder.DisallowUnknownFields()

		if err := newDecoder.Decode(&updateBody); err != nil {
			t.Errorf("Response structural validation failed: %v", err)
		}

		if updateBody.Recipes[0].Title != "new title" {
			t.Errorf("Failed update verification. expected title: 'new title' got: %s", updateBody.Recipes[0].Title)
		}
	})

	// delete recipe
	t.Run("delete recipe", func(t *testing.T) {
		url := "/api/recipes/" + testRecipeID.String()
		req = httptest.NewRequest("DELETE", url, nil)

		req.AddCookie(jwt)
		req.AddCookie(rt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Failed to delete recipe: got status %d", w.Code)
			return
		}

		// verify 404 status
		req = httptest.NewRequest("GET", url, nil)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Errorf("Expected 404: got status %d", w.Code)
			return
		}
	})
}

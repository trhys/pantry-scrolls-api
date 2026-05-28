package server_test

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "mime/multipart"
  "testing"
  "net/http"
  "net/http/httptest"
  "net/textproto"

  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/config"
  "github.com/aws/aws-sdk-go-v2/service/s3"
  "github.com/aws/aws-sdk-go-v2/credentials"
  "github.com/trhys/Recipe-Repo-2/internal/server"
  vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestCreateRecipe(t *testing.T) {
  cfg := server.GetConfig()
  tx, err := cfg.DBConn.Begin()
  if err != nil {
    t.Fatalf("Failed to start transaction: %v", err)
  }
  defer tx.Rollback()

  cfg.DB = cfg.DB.WithTx(tx)
  router := server.GetRouter(cfg)

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

  testIngredients := []struct {
      Name     string
      Quantity float64
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

  ingredientUUIDs := make(map[string]string)

  for _, ing := range testIngredients {
      id, err := cfg.DB.GetIngredientFromName(context.Background(), ing.Name)
      if err != nil {
          t.Fatalf("Failed to resolve UUID for ingredient %s: %v", ing.Name, err)
      }
      
      ingredientUUIDs[ing.Name] = id
  }

  recipePayload := fmt.Sprint(`
    {
	    "title": "Classic Spaghetti Bolognese",
	    "description": "A hearty Italian meat sauce slow-simmered with tomatoes and aromatics, served over al dente spaghetti.",
	    "ingredients": [
	      { "name": "Spaghetti", "quantity": 12, "unit": "Ounce" },
	      { "name": "Ground Beef", "quantity": 1, "unit": "Pound" },
	      { "name": "Yellow Onion", "quantity": 1, "unit": "Count" },
	      { "name": "Garlic", "quantity": 4, "unit": "Count" },
	      { "name": "Carrots", "quantity": 1, "unit": "Count" },
	      { "name": "Celery", "quantity": 2, "unit": "Count" },
	      { "name": "Crushed Tomatoes", "quantity": 1, "unit": "Cup" },
	      { "name": "Tomato Paste", "quantity": 2, "unit": "Tablespoon" },
	      { "name": "Extra Virgin Olive Oil", "quantity": 2, "unit": "Tablespoon" },
	      { "name": "Dried Oregano", "quantity": 1, "unit": "Teaspoon" },
	      { "name": "Kosher Salt", "quantity": 1, "unit": "Teaspoon" },
	      { "name": "Ground Black Pepper", "quantity": 0.5, "unit": "Teaspoon" },
	      { "name": "Parmesan Cheese", "quantity": 0.5, "unit": "Cup" }
	    ],
	    "instructions": "1. Bring a large pot of salted water to a boil. Cook spaghetti according to package directions until al dente; drain and set aside.\n2. Heat olive oil in a large skillet or Dutch oven over medium-high heat. Add ground beef and cook, breaking it apart, until browned, about 8 minutes. Drain excess fat.\n3. Add diced yellow onion, minced garlic, and diced carrot and celery to the pot. Cook, stirring frequently, until vegetables are softened, about 5 minutes.\n4. Stir in tomato paste and cook for 2 minutes until it darkens slightly.\n5. Add crushed tomatoes, dried oregano, salt, and pepper. Stir to combine.\n6. Reduce heat to low and simmer uncovered for 25–30 minutes, stirring occasionally, until the sauce has thickened.\n7. Toss the cooked spaghetti with the sauce and serve topped with grated Parmesan cheese."
	  }`)

  // Write multipart form data
  body := &bytes.Buffer{}
  writer := multipart.NewWriter(body)

  if err := writer.WriteField("payload", recipePayload); err != nil {
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

  // run
  t.Run("create recipe", func (t *testing.T) {
    req := httptest.NewRequest("POST", "/api/recipes", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    ctx := context.WithValue(req.Context(), "userID", user.ID)
    req = req.WithContext(ctx)

    req.AddCookie(jwt)
    req.AddCookie(rt)

    w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to create recipe: got status %d", w.Code)
			return
		}
    
    responseResult := w.Result()
    defer responseResult.Body.Close()

  	var responseBody vm.RecipeFull 
  	decoder := json.NewDecoder(responseResult.Body)
  	decoder.DisallowUnknownFields()
  
  	if err := decoder.Decode(&responseBody); err != nil {
  		t.Errorf("Response structural validation failed: %v", err)
  	}
  })
}

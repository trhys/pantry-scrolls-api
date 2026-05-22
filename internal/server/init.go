package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
  "github.com/joho/godotenv"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/data"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/config"
)

func GetRouter(cfg *apiConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// User eps
	mux.HandleFunc("GET /api/users/{user_id}", cfg.authMiddleware(cfg.handlerGetUserProfile))
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("POST /api/sessions", cfg.handlerLogin)
	mux.HandleFunc("GET /api/sessions", cfg.authMiddleware(cfg.handlerGetSession))
	mux.HandleFunc("PUT /api/users", cfg.authMiddleware(cfg.handlerUploadUserImage))

	// Recipe eps
	mux.HandleFunc("GET /api/recipes/{recipe_id}", cfg.handlerGetRecipe)
	mux.HandleFunc("GET /api/recipes", cfg.handlerGetRecipeList)
	mux.HandleFunc("POST /api/recipes", cfg.authMiddleware(cfg.handlerCreateRecipe))
	mux.HandleFunc("PUT /api/recipes/{recipe_id}", cfg.authMiddleware(cfg.handlerUpdateRecipe))
	mux.HandleFunc("DELETE /api/recipes/{recipe_id}", cfg.authMiddleware(cfg.handlerDeleteRecipe))
    mux.HandleFunc("GET /api/recipes/explore", cfg.handlerExploreFeed)

	// Ingredient eps
	//mux.HandleFunc("POST /api/ingredients", cfg.handlerCreateIngredient)
	mux.HandleFunc("GET /api/ingredients", cfg.handlerGetIngredientBase)
	mux.HandleFunc("GET /api/ingredients/{ingredient_id}/units", cfg.handlerGetUnits)

	// Shopping list eps
	mux.HandleFunc("GET /api/shoppinglists/{shopping_list_id}", cfg.authMiddleware(cfg.handlerGetShoppingList))
	mux.HandleFunc("GET /api/shoppinglists", cfg.authMiddleware(cfg.handlerGetUsersShoppingLists))
	mux.HandleFunc("POST /api/shoppinglists", cfg.authMiddleware(cfg.handlerCreateShoppingList))
	mux.HandleFunc("POST /api/shoppinglists/{shopping_list_id}", cfg.authMiddleware(cfg.handlerAddToShoppingList))
	mux.HandleFunc("GET /api/shoppinglists/{shopping_list_id}/print", cfg.authMiddleware(cfg.handlerPrintList))
	mux.HandleFunc("DELETE /api/shoppinglists/{shopping_list_id}", cfg.authMiddleware(cfg.handlerDeleteShoppingList))

	// Token eps
	mux.HandleFunc("GET /api/tokens/refresh", cfg.handlerRefreshToken)
	mux.HandleFunc("GET /api/tokens/revoke", cfg.handlerRevokeToken)

	return mux
}

func GetConfig() *apiConfig {
	godotenv.Load()

	dbUrl := os.Getenv("DB")
	if dbUrl == "" {
		log.Fatal("Failed to load database: url missing")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("Failed to load platform config")
	}

	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Fatal("Failed to load secret")
	}

	jwtDur := os.Getenv("JWT_DUR")
	if jwtDur == "" {
		log.Fatal("Failed to load jwt duration")
	}

	convDur, err := strconv.Atoi(jwtDur)
	if err != nil {
		log.Print("Failed to load jwt duration - defaulting to 3600")
		convDur = 3600
	}
	jwtDuration := time.Duration(convDur)*time.Second

	
	s3bucket := os.Getenv("S3_BUCKET")
	if s3bucket == "" {
		log.Fatal("Failed to load s3 bucket")
	}

	s3region := os.Getenv("S3_REGION")
	if err != nil {
		log.Fatal("Failed to load s3 region")
	}

	s3cdn := os.Getenv("S3_CDN")
	if s3cdn == "" {
		log.Fatal("Failed to load s3 CDN")
	}

	imagePlaceholder := os.Getenv("IMAGE_PLACEHOLDER")
	if imagePlaceholder == "" {
		log.Fatal("Failed to load placehold for images")
	}

	userpw := os.Getenv("USERPW")
	if userpw == "" {
		log.Fatal("Failed to get root user")
	}

	reacturl := os.Getenv("REACTURL")
	if reacturl == "" {
		log.Fatal("Failed to get frontend server")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Failed to load database: connection failed")
	}

	// Load S3 client
	s3cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3region))
	if err != nil {
		log.Fatal("Failed to load s3 config")
	}
	
	cfg := apiConfig{
		db: database.New(db),
		platform: platform,
		secret: secret,
		jwtDuration: jwtDuration,
		s3client: s3.NewFromConfig(s3cfg) ,
		s3bucket: s3bucket,
		s3region: s3region,
		s3cdn: s3cdn,
		imagePlaceholder: imagePlaceholder,
		vmf: viewmodel.VMFactory{
			DB: database.New(db),
			S3cdn: s3cdn,
		},
	}

	return cfg
}

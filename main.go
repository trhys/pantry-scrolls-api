package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rs/cors"
	_ "github.com/lib/pq"
        "github.com/joho/godotenv"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/data"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
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


	appDirectory := os.Getenv("APP_DIR")
	if appDirectory == "" {
		log.Fatal("Failed to load app directory")
	}

	adminDir := os.Getenv("ADMIN_DIR")
	if adminDir == "" {
		log.Fatal("Failed to load admin directory")
	}
	
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

	hash, _ := auth.HashPassword(userpw)

	// Connect to database
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Failed to load database: connection failed")
	}

	log.Print("Successfully loaded database...")

	// Load S3 client
	s3cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3region))
	if err != nil {
		log.Fatal("Failed to load s3 config")
	}

	log.Print("Successfully loaded s3 config...")

	// Load api config
	
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
			S3cdn: s3cdn,
		},
	}

	// Check database seeding
	if err := data.InitDBIngredients(imagePlaceholder, db, context.Background()); err != nil {
		log.Panic("Failed to seed database ingredients")
	}

	if err := data.InitDBRecipes(imagePlaceholder, db, context.Background(), hash); err != nil {
		log.Panic("Failed to seed database recipes")
	}
		
	// Load server
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	    	AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	    	AllowedHeaders: []string{"Authorization", "Content-Type", "Accept"},
	})

	mux := http.NewServeMux()
	server := http.Server{
		Addr: "0.0.0.0:8080",
		Handler: c.Handler(mux),
	}

	
	// Handlers :

	// User eps
	mux.HandleFunc("GET /users/{user_id}", cfg.authMiddleware(cfg.handlerGetUserProfile))
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("POST /api/sessions", cfg.handlerLogin)

	// Recipe eps
	mux.HandleFunc("GET /recipes/{recipe_id}", cfg.handlerGetRecipe)
	mux.HandleFunc("GET /api/recipes", cfg.handlerGetRecipeList)
	mux.HandleFunc("POST /api/recipes", cfg.authMiddleware(cfg.handlerCreateRecipe))
	mux.HandleFunc("UPDATE /api/recipes/{recipe_id}", cfg.authMiddleware(cfg.handlerUpdateRecipe))
	mux.HandleFunc("DELETE /api/recipes/{recipe_id}", cfg.authMiddleware(cfg.handlerDeleteRecipe))

	// Ingredient eps
	//mux.HandleFunc("POST /api/ingredients", cfg.handlerCreateIngredient)
	mux.HandleFunc("GET /api/ingredients", cfg.handlerGetIngredientBase)
	mux.HandleFunc("POST /api/units", cfg.handlerGetUnits)

	// Shopping list eps
	mux.HandleFunc("GET /shoppinglists/{shopping_list_id}", cfg.authMiddleware(cfg.handlerGetShoppingList))
	mux.HandleFunc("GET /users/{user_id}/shoppinglists", cfg.authMiddleware(cfg.handlerGetUsersShoppingLists))
	mux.HandleFunc("POST /api/shoppinglists", cfg.authMiddleware(cfg.handlerCreateShoppingList))
	mux.HandleFunc("POST /api/shoppinglists/{shopping_list_id}", cfg.authMiddleware(cfg.handlerAddToShoppingList))
	mux.HandleFunc("GET /shoppinglists/{shopping_list_id}/print", cfg.authMiddleware(cfg.handlerPrintList))
	mux.HandleFunc("DELETE /api/shoppinglists/{shopping_list_id}", cfg.authMiddleware(cfg.handlerDeleteShoppingList))

	// Token eps
	mux.HandleFunc("POST /api/tokens/refresh", cfg.handlerRefreshToken)
	mux.HandleFunc("POST /api/tokens/revoke", cfg.handlerRevokeToken)

	// :

	log.Print("Successfully loaded server...")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

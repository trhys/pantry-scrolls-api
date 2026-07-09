package server

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func GetRouter(cfg *ApiConfig, reg *prometheus.Registry) *http.ServeMux {
	mux := http.NewServeMux()

	// Metrics ep
	mux.Handle("/api/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// User eps
	mux.HandleFunc("GET /api/users/{user_id}", cfg.authMiddleware(cfg.handlerGetUserProfile))
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("POST /api/sessions", cfg.handlerLogin)
	mux.HandleFunc("GET /api/sessions", cfg.authMiddleware(cfg.handlerGetSession))
	mux.HandleFunc("PUT /api/users", cfg.authMiddleware(cfg.handlerUploadUserImage))
	mux.HandleFunc("PUT /api/users/{user_id}", cfg.authMiddleware(cfg.handlerUpdateUser))
	mux.HandleFunc("PUT /api/users/{user_id}/deactivate", cfg.authMiddleware(cfg.handlerDeactivateUser))
	mux.HandleFunc("PUT /api/deactivation/cancel", cfg.handlerCancelDeactivation)
	mux.HandleFunc("GET /api/verify/{token}", cfg.handlerVerifyEmail)
	mux.HandleFunc("POST /api/resetpassword", cfg.handlerResetPassword)
	mux.HandleFunc("PUT /api/resetpassword", cfg.handlerUpdatePassword)

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

	// Messages eps
	mux.HandleFunc("POST /api/messages", cfg.handlerAddMessage)
	mux.HandleFunc("GET /api/messages", cfg.handlerGetMessages)
	mux.HandleFunc("POST /api/messages/{message_id}", cfg.handlerChangeMessageStatus)

	return mux
}

func GetConfig() *ApiConfig {
	godotenv.Load()

	dbUrl := os.Getenv("DB")
	if dbUrl == "" {
		slog.Error("Environment load failure", "missing", "DB")
		os.Exit(1)
	}

	secret := os.Getenv("SECRET")
	if secret == "" {
		slog.Error("Environment load failure", "missing", "SECRET")
		os.Exit(1)
	}

	jwtDur := os.Getenv("JWT_DUR")
	if jwtDur == "" {
		slog.Error("Environment load failure", "missing", "JWT_DUR")
		os.Exit(1)
	}

	convDur, err := strconv.Atoi(jwtDur)
	if err != nil {
		slog.Warn("Failed to load jwt duration - defaulting to 3600")
		convDur = 3600
	}
	jwtDuration := time.Duration(convDur) * time.Second

	s3bucket := os.Getenv("S3_BUCKET")
	if s3bucket == "" {
		slog.Error("Environment load failure", "missing", "S3_BUCKET")
		os.Exit(1)
	}

	s3region := os.Getenv("S3_REGION")
	if err != nil {
		slog.Error("Environment load failure", "missing", "S3_REGION")
		os.Exit(1)
	}

	s3cdn := os.Getenv("S3_CDN")
	if s3cdn == "" {
		slog.Error("Environment load failure", "missing", "S3_CDN")
		os.Exit(1)
	}

	imagePlaceholder := os.Getenv("IMAGE_PLACEHOLDER")
	if imagePlaceholder == "" {
		slog.Error("Environment load failure", "missing", "IMAGE_PLACEHOLDER")
		os.Exit(1)
	}

	userpw := os.Getenv("USERPW")
	if userpw == "" {
		slog.Error("Environment load failure", "missing", "USERPW")
		os.Exit(1)
	}

	reactUrl := os.Getenv("REACTURL")
	if reactUrl == "" {
		slog.Error("Environment load failure", "missing", "REACTURL")
		os.Exit(1)
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKey == "" {
		slog.Error("Environment load failure", "missing", "AWS_ACCESS_KEY_ID")
		os.Exit(1)
	}

	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretKey == "" {
		slog.Error("Environment load failure", "missing", "AWS_SECRET_ACCESS_KEY")
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		slog.Error("Failed to establish db connection", "error", err)
		os.Exit(1)
	}

	// set db connection pool TODO: benchmark and adjust if needed
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Load S3 cfg
	s3cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3region))
	if err != nil {
		slog.Error("Failed to load s3 config", "error", err)
	}

	// Load SES cfg
	sesCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3region))
	if err != nil {
		slog.Error("Failed to load SES config", "error", err)
	}

	cfg := ApiConfig{
		DB:               database.New(db),
		DBConn:           db,
		Secret:           secret,
		JwtDuration:      jwtDuration,
		SESClient:        ses.NewFromConfig(sesCfg),
		S3client:         s3.NewFromConfig(s3cfg),
		S3bucket:         s3bucket,
		S3region:         s3region,
		S3cdn:            s3cdn,
		ImagePlaceholder: imagePlaceholder,
		Vmf: viewmodel.VMFactory{
			DB:    database.New(db),
			S3cdn: s3cdn,
		},
		Root: adminCredentials{
			Email: "recipereporoot@admin.trr",
			Pass:  userpw,
		},
		React: reactUrl,
	}

	return &cfg
}

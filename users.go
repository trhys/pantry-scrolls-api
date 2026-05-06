package main

import (
	"path/filepath"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	util "github.com/trhys/Recipe-Repo-2/internal/utility"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	// Decode request
	var req struct {
		Email		string `json:"email"`
		Password	string `json:"password"`
		Name		string `json:"name"`
	}

	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(w, 400, "Bad request", fmt.Errorf("Failed to decode request - ERROR: %v", err))
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to hash password for user email: %s - ERROR: %v", req.Email, err))
		return
	}

	query := database.CreateUserParams{
		Email: req.Email,
		HashedPw: hash,
		Name: req.Name,
	}

	user, err := cfg.db.CreateUser(r.Context(), query)
	if err != nil {
		respondFail(w, 500, "Database error", fmt.Errorf("Failed to perform CreateUser query: %v", err))
		return
	}

	log.Printf("User created with email: %s", user.Email)
	respondJSON(w, 204, nil)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email		string `json:"email"`
		Password	string `json:"password"`
	}

	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(w, 400, "Bad request", fmt.Errorf("Failed to decode request - ERROR: %v", err))
		return
	}

	// get users info
	user, err := cfg.db.GetUserHash(r.Context(), req.Email)
	if err != nil {
		respondFail(w, 404, "User not found", fmt.Errorf("Failed to find user with email: %s - ERROR: %v", req.Email, err))
		return
	}

	// check the hash
	match, err := auth.CheckPasswordHash(req.Password, user.HashedPw)
	if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to check hash: %v", err))
		return
	}

	if match {
		token, err := auth.MakeJWT(user.ID, cfg.secret, cfg.jwtDuration)
		if err != nil{
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to write JWT for user: %s, - ERROR: %v", req.Email, err))
			return
		}

		refreshToken := auth.MakeRefreshToken()
		if _, err := cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			ID: refreshToken,
			UserID: user.ID,
		}); err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to generate refresh token for user: %s - ERROR: %v", req.Email, err))
			return
		}

		cookie := http.Cookie{
			Name:     "jwt",
			Value:    token,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
			Expires:  time.Now().Add(1 * time.Hour),
		}
		
		http.SetCookie(w, &cookie)

		respondJSON(w, 200, viewmodel.GenerateSession(user, token, refreshToken)) 
		return
	} else {
		respondFail(w, 401, "Invalid username or password", fmt.Errorf("Failed login attempt for: %s", user.Email))
		return
	}
}

func (cfg *apiConfig) handlerGetUserProfile(w http.ResponseWriter, r *http.Request) {
	// Get user id from path
	val := r.PathValue("user_id")
        id, err := uuid.Parse(val)
        if err != nil {
		respondFail(w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID in url: %v", err))
                return
        }

	// Make sure user exists
        user, err := cfg.db.GetUser(r.Context(), id)
        if err != nil {
		respondFail(w, 404, "Couldn't find user", fmt.Errorf("Failed to find user with ID: %s, ERROR: %v", val, err))
                return
        }

	// Validate auth from middleware
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", val))
		return
	}

	// Get recipes for user
        recipes, err := cfg.db.GetUsersRecipes(r.Context(), user.ID)
        if err != nil {
		respondFail(w, 404, "Couldn't find recipes", fmt.Errorf("Failed to find recipes for user ID: %s - ERROR: %v", val, err))
                return
        }

	// Branch on public/private view based on whether the requester is the user being requested
	var viewModel any
	if requesterID == user.ID {
		viewModel = cfg.vmf.GeneratePrivateUser(user, recipes)
	} else {
		viewModel = cfg.vmf.GeneratePublicUser(user, recipes)
	}

        if r.Header.Get("Accept") == "application/json" {
                respondJSON(w, 200, viewModel)
                return
        }

        tmpl, err := template.ParseFiles(filepath.Join("app", "templates", "user_page.html"))
        if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to render HTML template: %v", err))
                return
        }

        tmpl.Execute(w, viewModel)
}

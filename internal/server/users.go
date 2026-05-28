package server

import (
	"fmt"
	"io"
	"os"
	"log"
	"mime"
	"net/http"
    "net/mail"
    "strings"
	"time"

	"github.com/lib/pq"
    "github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	util "github.com/trhys/Recipe-Repo-2/internal/utility"
	_ "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func (cfg *ApiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
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

    // Verify valid email address
    if _, err := mail.ParseAddress(req.Email); err != nil {
      respondFail(w, 400, "Invalid email address", fmt.Errorf("Bad email in create user request: %v", err))
      return
    }

	// Enforce case insensitivity
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

    // Verify password length
    if len(req.Password) < 5 {
      respondFail(w, 400, "Password too short", fmt.Errorf("Password too short (Create User)"))
      return
    }

    // Verify user name length
    if len(req.Name) > 30 {
      respondFail(w, 400, "Username too long", fmt.Errorf("Username too long (Create User)"))
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

	user, err := cfg.DB.CreateUser(r.Context(), query)
	if err != nil {
      if err.(*pq.Error).Code  == "23505" {
        respondFail(w, 400, "Email address is already associated with a user account!", fmt.Errorf("Duplicate user query: %v", err))
        return
      }
		respondFail(w, 500, "Database error", fmt.Errorf("Failed to perform CreateUser query: %v", err))
		return
	}

	log.Printf("User created with email: %s", user.Email)
	respondJSON(w, 201, nil)
}

func (cfg *ApiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email		string `json:"email"`
		Password	string `json:"password"`
	}

	if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
		respondFail(w, 400, "Bad request", fmt.Errorf("Failed to decode request - ERROR: %v", err))
		return
	}

	// Validate fields
	if req.Email == "" || req.Password == "" {
		respondFail(w, 400, "Missing email or password", fmt.Errorf("Bad request missing email or password (Login)"))
		return
	}

	// Enforce case insensitivity
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// get users info
	user, err := cfg.DB.GetUserHash(r.Context(), req.Email)
	if err != nil {
		respondFail(w, 401, "Invalid email or password", fmt.Errorf("Failed to find user with email: %s - ERROR: %v", req.Email, err))
		return
	}

	// check the hash
	match, err := auth.CheckPasswordHash(req.Password, user.HashedPw)
	if err != nil {
		respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to check hash: %v", err))
		return
	}

	if match {
		token, err := auth.MakeJWT(user.ID, cfg.Secret, cfg.JwtDuration)
		if err != nil{
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to write JWT for user: %s, - ERROR: %v", req.Email, err))
			return
		}

		refreshToken := auth.MakeRefreshToken()
		if _, err := cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
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
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  time.Now().Add(1 * time.Hour),
		}
		
		http.SetCookie(w, &cookie)

        refreshCookie := http.Cookie{
          Name:     "refresh_token",
          Value:    refreshToken,
          HttpOnly: true,
          Secure:   true,
          SameSite: http.SameSiteLaxMode,
          Path:     "/",
          Expires:  time.Now().Add(30 * (24 * time.Hour)),
        }

        http.SetCookie(w, &refreshCookie)

		respondJSON(w, 200, cfg.Vmf.GenerateSession(user, token, refreshToken)) 
		return
	} else {
		respondFail(w, 401, "Invalid username or password", fmt.Errorf("Failed login attempt for: %s", user.Email))
		return
	}
}

// Refresh user session
func (cfg *ApiConfig) handlerGetSession(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", id))
		return
	}

	user, err := cfg.DB.RefreshUser(r.Context(), id)
	if err != nil {
		respondFail(w, 404, "Couldn't find user", fmt.Errorf("Database query failed (RefreshUser) : %v", err))
		return
	}

	respondJSON(w, 200, cfg.Vmf.RefreshSession(user))
}

func (cfg *ApiConfig) handlerGetUserProfile(w http.ResponseWriter, r *http.Request) {
	// Get user id from path
	val := r.PathValue("user_id")
        id, err := uuid.Parse(val)
        if err != nil {
		respondFail(w, 404, "Invalid uuid", fmt.Errorf("Failed to parse UUID in url: %v", err))
                return
        }

	// Make sure user exists
        user, err := cfg.DB.GetUser(r.Context(), id)
        if err != nil {
		respondFail(w, 404, "Couldn't find user", fmt.Errorf("Failed to find user with ID: %s, ERROR: %v", val, err))
                return
        }

	// Validate auth from middleware
	requesterID := r.Context().Value("userID")
    //_, ok := requesterID.(uuid.UUID)
	//if !ok {
	//	respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", val))
	//	return
	//}

	// Get recipes for user
        recipes, err := cfg.DB.GetUsersRecipes(r.Context(), user.ID)
        if err != nil {
		respondFail(w, 404, "Couldn't find recipes", fmt.Errorf("Failed to find recipes for user ID: %s - ERROR: %v", val, err))
                return
        }

	// Branch on public/private view based on whether the requester is the user being requested
	var viewModel any
	if requesterID == user.ID {
		viewModel = cfg.Vmf.GeneratePrivateUser(user, recipes)
	} else {
		viewModel = cfg.Vmf.GeneratePublicUser(user, recipes)
	}

        respondJSON(w, 200, viewModel)
}

// Upload user profile image
func (cfg *ApiConfig) handlerUploadUserImage(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt at user id: %s", requesterID))
		return
	}

        key, err := cfg.DB.GetUserImageKey(r.Context(), requesterID)
	if err != nil {
		respondFail(w, 404, "Couldn't get user image key", fmt.Errorf("Failed to get user image key. UserID: %s - ERROR: %v", requesterID, err))
		return
	}

	firstUpload := false
	if key == "" {
		key = "/users" + uuid.New().String()
		firstUpload = true
	}

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

	// If this is the first upload, set image key in user database
	if firstUpload {
		if err := cfg.DB.SetUserImageKey(r.Context(), database.SetUserImageKeyParams{
			ID: requesterID,
			ImageKey: key,
		}); err != nil {
			respondFail(w, 500, "Something went wrong", fmt.Errorf("Failed to set key in user database: %v", err))
			return
		}
	}

	respondJSON(w, 204, nil)
}

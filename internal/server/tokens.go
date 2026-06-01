package server

import (
	"net/http"
    "time"

	"github.com/trhys/Recipe-Repo-2/internal/auth"
)

func (cfg *ApiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
    var tokenString string
    token, err := r.Cookie("refresh_token")
    if err == nil {
      tokenString = token.Value
    }

    if tokenString == "" {
      token, err := auth.GetBearerToken(r.Header)
	  if err != nil {
	  	  respondFail(w, 401, "Invalid header", err)
		  return
	  } else {
        tokenString = token
      }
    }

	user, err := cfg.DB.GetRefreshToken(r.Context(), tokenString)
	if err != nil {
		respondFail(w, 401, "Invalid token", err)
		return
	}
	
	jwt, err := auth.MakeJWT(user, cfg.Secret, cfg.JwtDuration)
	if err != nil {
		respondFail(w, 500, "Failed to write JWT", err)
		return
	}

	type res struct{
		Token string `json:"token"`
	}

	resp := res{
		Token: jwt,
	}

    jwtCookie := http.Cookie{
      Name: "jwt",
      Value: jwt,
      HttpOnly: true,
      Secure:   true,
      SameSite: http.SameSiteLaxMode,
      Path:     "/",
      Expires:  time.Now().Add(1 * time.Hour),
    }

    http.SetCookie(w, &jwtCookie)

	respondJSON(w, 200, resp)
}

func (cfg *ApiConfig) handlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	var tokenString string
    token, err := r.Cookie("refresh_token")
    if err == nil {
      tokenString = token.Value
    }

    if tokenString == "" {
      token, err := auth.GetBearerToken(r.Header)
	  if err != nil {
	  	  respondFail(w, 401, "Invalid header", err)
		  return
	  } else {
        tokenString = token
      }
    }

	if err := cfg.DB.RevokeToken(r.Context(), tokenString); err != nil {
		respondFail(w, 401, "Invalid token", err)
		return
	}

    jwtCookie := http.Cookie{
      Name: "jwt",
      Value: "",
      Path: "/",
      HttpOnly: true,
      Secure: true,
      SameSite: http.SameSiteLaxMode,
      MaxAge: -1,
      Expires: time.Unix(0,0),
    }

    rtCookie := http.Cookie{
      Name: "refresh_token",
      Value: "",
      Path: "/",
      HttpOnly: true,
      Secure: true,
      SameSite: http.SameSiteLaxMode,
      MaxAge: -1,
      Expires: time.Unix(0,0),
    }

    http.SetCookie(w, &jwtCookie)
    http.SetCookie(w, &rtCookie)
  
	respondJSON(w, 204, nil)
}

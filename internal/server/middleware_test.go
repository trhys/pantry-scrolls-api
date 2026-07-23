package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
)

func TestOptionalAuthMiddleware(t *testing.T) {
	cfg := &ApiConfig{Secret: "test-secret"}

	t.Run("injects requester when token is valid", func(t *testing.T) {
		userID := uuid.New()
		token, err := auth.MakeJWT(userID, cfg.Secret, time.Hour)
		if err != nil {
			t.Fatalf("failed to make jwt: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
		req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
		rec := httptest.NewRecorder()

		cfg.optionalAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			requesterID, ok := requesterUserID(r)
			if !ok {
				t.Fatalf("expected requester id in context")
			}
			if requesterID != userID {
				t.Fatalf("expected requester %s, got %s", userID, requesterID)
			}
			w.WriteHeader(http.StatusNoContent)
		}).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("keeps request anonymous when token missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
		rec := httptest.NewRecorder()

		cfg.optionalAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := requesterUserID(r); ok {
				t.Fatalf("did not expect requester id in context")
			}
			if r.Context().Value("userID") != "" {
				t.Fatalf("expected anonymous user context")
			}
			w.WriteHeader(http.StatusNoContent)
		}).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("ignores invalid token on public route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
		req.AddCookie(&http.Cookie{Name: "jwt", Value: "not-a-valid-token"})
		rec := httptest.NewRecorder()

		cfg.optionalAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := requesterUserID(r); ok {
				t.Fatalf("did not expect requester id in context")
			}
			w.WriteHeader(http.StatusNoContent)
		}).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
		}
	})
}

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestAuthenticationEdgeCases(t *testing.T) {
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

	testUser := struct {
		input []byte
	}{
		input: []byte(`{"email":"edge@test.com", "password": "password", "name": "edge"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Login test user
	login := []byte(`{"email":"edge@test.com", "password": "password"}`)
	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var user vm.SessionViewModel
	json.NewDecoder(w.Body).Decode(&user)

	t.Run("access protected endpoint without auth", func(t *testing.T) {
		url := "/api/users/" + user.ID.String()
		req := httptest.NewRequest("GET", url, nil)
		// No cookies or headers

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still get 200 because it's a public view
		if w.Code != 200 {
			t.Errorf("Expected 200 for public user view: got status %d", w.Code)
		}
	})

	t.Run("access private endpoint without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sessions", nil)
		// No cookies or headers

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return error since endpoint requires auth
		if w.Code >= 200 && w.Code < 300 {
			t.Errorf("Expected error status for unauthenticated private endpoint: got status %d", w.Code)
		}
	})

	t.Run("access another user's private data", func(t *testing.T) {
		// Create second user
		testUser2 := struct {
			input []byte
		}{
			input: []byte(`{"email":"edge2@test.com", "password": "password", "name": "edge2"}`),
		}

		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser2.input))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Login first user and get auth
		login := []byte(`{"email":"edge@test.com", "password": "password"}`)
		req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var user1 vm.SessionViewModel
		json.NewDecoder(w.Body).Decode(&user1)

		response := w.Result()
		cookies := response.Cookies()
		var jwt *http.Cookie
		for _, c := range cookies {
			if c.Name == "jwt" {
				jwt = c
			}
		}

		// Get user2's ID
		user2DB, err := cfg.DB.GetUserByEmail(context.Background(), "edge2@test.com")
		if err != nil {
			t.Fatalf("Failed to get user2: %v", err)
		}

		// Try to access user2's profile with user1's auth
		url := "/api/users/" + user2DB.ID.String()
		req = httptest.NewRequest("GET", url, nil)
		req.AddCookie(jwt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should succeed since user profile is public
		if w.Code != 200 {
			t.Errorf("Expected 200 for public user profile: got status %d", w.Code)
		}

		var publicUser vm.PublicUserViewModel
		if err := json.NewDecoder(w.Body).Decode(&publicUser); err != nil {
			t.Errorf("Failed to decode public user view: %v", err)
		}

		// Verify no private email is exposed
		if publicUser.Email != "" {
			t.Errorf("Security leak: email exposed in public view: %s", publicUser.Email)
		}
	})
}

func TestMalformedRequests(t *testing.T) {
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

	t.Run("login with malformed json", func(t *testing.T) {
		body := []byte(`{"email": "test@test.com", "password": "pass"`) // Missing closing brace
		req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected 400 for malformed JSON: got status %d", w.Code)
		}
	})

	t.Run("create user with null values", func(t *testing.T) {
		body := []byte(`{"email": null, "password": "pass", "name": "test"}`)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected 400 for null values: got status %d", w.Code)
		}
	})

	t.Run("get user with invalid UUID format", func(t *testing.T) {
		url := "/api/users/not-a-uuid"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for invalid UUID: got status %d", w.Code)
		}
	})

	t.Run("get user with valid UUID format but nonexistent", func(t *testing.T) {
		fakeUUID := uuid.New()
		url := "/api/users/" + fakeUUID.String()
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for nonexistent user: got status %d", w.Code)
		}
	})
}

func TestAuthorizationErrors(t *testing.T) {
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

	t.Run("update user with wrong user id", func(t *testing.T) {
		// Create user
		testUser := struct {
			input []byte
		}{
			input: []byte(`{"email":"auth@test.com", "password": "password", "name": "auth"}`),
		}

		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Login
		login := []byte(`{"email":"auth@test.com", "password": "password"}`)
		req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var user vm.SessionViewModel
		json.NewDecoder(w.Body).Decode(&user)

		response := w.Result()
		cookies := response.Cookies()
		var jwt *http.Cookie
		for _, c := range cookies {
			if c.Name == "jwt" {
				jwt = c
			}
		}

		// Try to update different user's profile
		fakeUserID := uuid.New()
		url := "/api/users/" + fakeUserID.String()
		updateBody := []byte(`{"name":"hacker"}`)
		req = httptest.NewRequest("PUT", url, bytes.NewBuffer(updateBody))
		req.AddCookie(jwt)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with 401 or similar
		if w.Code >= 200 && w.Code < 300 {
			t.Errorf("Expected error for unauthorized update: got status %d", w.Code)
		}
	})
}

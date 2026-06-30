package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestTokenRefresh(t *testing.T) {
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

	t.Run("refresh with valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tokens/refresh", nil)
		req.AddCookie(rt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to refresh token: got status %d", w.Code)
			return
		}

		var response struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("failed to decode response: %v", err)
		}

		if response.Token == "" {
			t.Errorf("expected token in response")
		}
	})

	t.Run("refresh with bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tokens/refresh", nil)
		req.Header.Set("Authorization", "Bearer "+rt.Value)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to refresh token with bearer: got status %d", w.Code)
		}
	})

	t.Run("refresh with invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tokens/refresh", nil)
		invalidCookie := &http.Cookie{
			Name:  "refresh_token",
			Value: "invalid-token-12345",
		}
		req.AddCookie(invalidCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Errorf("Expected 401 for invalid token: got status %d", w.Code)
		}
	})

	t.Run("refresh with missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tokens/refresh", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Errorf("Expected 401 for missing token: got status %d", w.Code)
		}
	})
}

func TestTokenRevoke(t *testing.T) {
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
		input: []byte(`{"email":"tim@test.com", "password": "password", "name": "tim"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Login test user
	login := []byte(`{"email":"tim@test.com", "password": "password"}`)
	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var user vm.SessionViewModel
	json.NewDecoder(w.Body).Decode(&user)

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

	t.Run("revoke with valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tokens/revoke", nil)
		req.AddCookie(rt)
		req.AddCookie(jwt)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Failed to revoke token: got status %d", w.Code)
		}

		// Verify token is revoked by attempting to refresh
		req = httptest.NewRequest("GET", "/api/tokens/refresh", nil)
		req.AddCookie(rt)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Errorf("Expected 401 after revocation: got status %d", w.Code)
		}
	})

	t.Run("revoke with bearer token", func(t *testing.T) {
		// Create fresh session for this test
		login := []byte(`{"email":"tim@test.com", "password": "password"}`)
		req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var user vm.SessionViewModel
		json.NewDecoder(w.Body).Decode(&user)

		response := w.Result()
		cookies := response.Cookies()
		var rt *http.Cookie
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				rt = c
			}
		}

		req = httptest.NewRequest("GET", "/api/tokens/revoke", nil)
		req.Header.Set("Authorization", "Bearer "+rt.Value)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Errorf("Failed to revoke with bearer: got status %d", w.Code)
		}
	})

	t.Run("revoke with invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tokens/revoke", nil)
		invalidCookie := &http.Cookie{
			Name:  "refresh_token",
			Value: "invalid-token-xyz",
		}
		req.AddCookie(invalidCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Errorf("Expected 401 for invalid revoke: got status %d", w.Code)
		}
	})
}

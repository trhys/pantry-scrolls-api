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
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/server"
)

func TestEmailVerification(t *testing.T) {
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
		input: []byte(`{"email":"verify@test.com", "password": "password", "name": "verify user"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("Failed to create user for test")
	}

	// Verify mock SES was called
	if mockSES.CallCount == 0 {
		t.Errorf("Expected SES to be called for verification email")
	}

	if mockSES.LastToAddress != "verify@test.com" {
		t.Errorf("Expected verification email to verify@test.com, got %s", mockSES.LastToAddress)
	}

	t.Run("verify with valid token", func(t *testing.T) {
		// Create a verification token
		err := cfg.DB.CreateVerification(context.Background(), database.CreateVerificationParams{
			Email:     "verify@test.com",
			Token:     "valid-token-12345",
			ExpiresAt: time.Now().Add(30 * time.Minute),
		})
		if err != nil {
			t.Fatalf("Failed to create verification token: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/verify/"+token.Token, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Failed to verify email: got status %d", w.Code)
		}
	})

	t.Run("verify with invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/verify/invalid-token-xyz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for invalid token: got status %d", w.Code)
		}
	})

	t.Run("verify with expired token", func(t *testing.T) {
		// Create an expired verification token
		err := cfg.DB.CreateVerification(context.Background(), database.CreateVerificationParams{
			Email:     "verify@test.com",
			Token:     "expired-token-12345",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		})
		if err != nil {
			t.Fatalf("Failed to create verification token: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/verify/"+token.Token, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("Expected 401 for expired token: got status %d", w.Code)
		}
	})
}

func TestPasswordReset(t *testing.T) {
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
		input: []byte(`{"email":"reset@test.com", "password": "password", "name": "reset user"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("Failed to create user for test")
	}

	t.Run("request password reset for valid email", func(t *testing.T) {
		body := []byte(`{"email":"reset@test.com"}`)
		req := httptest.NewRequest("POST", "/api/resetpassword", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Failed to request password reset: got status %d", w.Code)
		}

		if mockSES.CallCount == 0 {
			t.Errorf("Expected SES to be called for reset email")
		}
	})

	t.Run("request password reset for nonexistent email", func(t *testing.T) {
		body := []byte(`{"email":"nonexistent@test.com"}`)
		req := httptest.NewRequest("POST", "/api/resetpassword", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 404 or 401
		if w.Code < 400 || w.Code >= 500 {
			t.Errorf("Expected 4xx status for nonexistent email: got status %d", w.Code)
		}
	})

	t.Run("request password reset with invalid email", func(t *testing.T) {
		body := []byte(`{"email":"not-an-email"}`)
		req := httptest.NewRequest("POST", "/api/resetpassword", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected 400 for invalid email: got status %d", w.Code)
		}
	})
}

func TestUpdatePassword(t *testing.T) {
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
		input: []byte(`{"email":"password@test.com", "password": "oldpassword", "name": "password user"}`),
	}

	// Create test user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Run("update password with valid token", func(t *testing.T) {
		// Create a reset token
		err := cfg.DB.CreateVerification(context.Background(), database.CreateVerificationTokenParams{
			Email:     "password@test.com",
			Token:     "reset-token-12345",
			ExpiresAt: time.Now().Add(30 * time.Minute),
		})
		if err != nil {
			t.Fatalf("Failed to create reset token: %v", err)
		}

		body := []byte(`{"token":"` + token.Token + `","password":"newpassword123"}`)
		req := httptest.NewRequest("PUT", "/api/resetpassword", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 204 {
			t.Errorf("Failed to update password: got status %d", w.Code)
		}
	})

	t.Run("update password with password too short", func(t *testing.T) {
		// Create a reset token
		err := cfg.DB.CreateVerification(context.Background(), database.CreateVerificationParams{
			Email:     "password@test.com",
			Token:     "reset-token-short",
			ExpiresAt: time.Now().Add(30 * time.Minute),
		})
		if err != nil {
			t.Fatalf("Failed to create reset token: %v", err)
		}

		body := []byte(`{"token":"` + token.Token + `","password":"abc"}`)
		req := httptest.NewRequest("PUT", "/api/resetpassword", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected 400 for short password: got status %d", w.Code)
		}
	})

	t.Run("update password with invalid token", func(t *testing.T) {
		body := []byte(`{"token":"invalid-token-xyz","password":"newpassword123"}`)
		req := httptest.NewRequest("PUT", "/api/resetpassword", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("Expected 404 for invalid token: got status %d", w.Code)
		}
	})
}

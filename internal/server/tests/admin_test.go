package server_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
)

func TestAdminCheck(t *testing.T) {
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

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("Expected 401 for unauthenticated request: got status %d", w.Code)
		}
	})

	t.Run("authenticated non-admin returns 403", func(t *testing.T) {
		// Create a regular user
		createUser := []byte(`{"email":"admincheck@test.com","password":"password","name":"admincheck"}`)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(createUser))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("Failed to create user: got status %d", w.Code)
		}

		// Login
		login := []byte(`{"email":"admincheck@test.com","password":"password"}`)
		req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("Failed to login: got status %d", w.Code)
		}

		response := w.Result()
		cookies := response.Cookies()
		var jwtCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "jwt" {
				jwtCookie = c
			}
		}

		req = httptest.NewRequest("GET", "/api/admin/check", nil)
		req.AddCookie(jwtCookie)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Errorf("Expected 403 for non-admin user: got status %d", w.Code)
		}
	})

	t.Run("authenticated admin returns 200", func(t *testing.T) {
		// Create a user and promote to admin
		createUser := []byte(`{"email":"admincheckadmin@test.com","password":"password","name":"admincheckadmin"}`)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(createUser))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("Failed to create admin user: got status %d", w.Code)
		}

		adminUser, err := cfg.DB.GetUserByEmail(context.Background(), "admincheckadmin@test.com")
		if err != nil {
			t.Fatalf("Failed to get admin user: %v", err)
		}

		if err := cfg.DB.MakeAdmin(context.Background(), adminUser.ID); err != nil {
			t.Fatalf("Failed to promote user to admin: %v", err)
		}

		// Login
		login := []byte(`{"email":"admincheckadmin@test.com","password":"password"}`)
		req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("Failed to login admin: got status %d", w.Code)
		}

		response := w.Result()
		cookies := response.Cookies()
		var jwtCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "jwt" {
				jwtCookie = c
			}
		}

		req = httptest.NewRequest("GET", "/api/admin/check", nil)
		req.AddCookie(jwtCookie)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected 200 for admin user: got status %d", w.Code)
		}
	})
}

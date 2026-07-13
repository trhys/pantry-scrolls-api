package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestUserDeactivationLifecycle(t *testing.T) {
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

	createUser := []byte(`{"email":"tim@test.com","password":"password","name":"tim"}`)
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(createUser))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("Failed to create user for test: got status %d", w.Code)
	}

	login := []byte(`{"email":"tim@test.com","password":"password"}`)
	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("Failed to login test user: got status %d", w.Code)
	}

	var user vm.SessionViewModel
	if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

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
	if jwt == nil || rt == nil {
		t.Fatal("expected login response to include jwt and refresh_token cookies")
	}

	req = httptest.NewRequest("PUT", "/api/users/"+user.ID.String()+"/deactivate", nil)
	req.AddCookie(jwt)
	req.AddCookie(rt)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("Failed to deactivate test user: got status %d", w.Code)
	}

	var deactivatedAt sql.NullTime
	if err := tx.QueryRowContext(context.Background(), "SELECT deactivated_at FROM users WHERE id = $1", user.ID).Scan(&deactivatedAt); err != nil {
		t.Fatalf("failed to query deactivated user: %v", err)
	}
	if !deactivatedAt.Valid {
		t.Fatal("expected user to be marked as deactivated")
	}

	var cancelToken string
	if err := tx.QueryRowContext(context.Background(), "SELECT token FROM deactivation_tokens WHERE user_id = $1", user.ID).Scan(&cancelToken); err != nil {
		t.Fatalf("failed to query deactivation token: %v", err)
	}

	if mockSES.CallCount != 2 {
		t.Fatalf("expected verification and deactivation emails to be sent, got %d calls", mockSES.CallCount)
	}
	if mockSES.LastToAddress != user.Email {
		t.Fatalf("expected deactivation email to be sent to %s, got %s", user.Email, mockSES.LastToAddress)
	}

	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected deactivated user login to fail with 401, got %d", w.Code)
	}

	cancelBody := []byte(`{"token":"` + cancelToken + `"}`)
	req = httptest.NewRequest("PUT", "/api/deactivation/cancel", bytes.NewBuffer(cancelBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("Failed to cancel deactivation: got status %d", w.Code)
	}

	if err := tx.QueryRowContext(context.Background(), "SELECT deactivated_at FROM users WHERE id = $1", user.ID).Scan(&deactivatedAt); err != nil {
		t.Fatalf("failed to re-query user after reactivation: %v", err)
	}
	if deactivatedAt.Valid {
		t.Fatal("expected user deactivation to be cleared")
	}

	var tokenCount int
	if err := tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM deactivation_tokens WHERE user_id = $1", user.ID).Scan(&tokenCount); err != nil {
		t.Fatalf("failed to count deactivation tokens: %v", err)
	}
	if tokenCount != 0 {
		t.Fatalf("expected deactivation token to be deleted, found %d", tokenCount)
	}

	req = httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(login))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected reactivated user login to succeed, got %d", w.Code)
	}
}

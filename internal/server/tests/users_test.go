package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/trhys/Recipe-Repo-2/internal/server"
	vm "github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

func TestCreateUser(t *testing.T) {
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

	tests := map[string]struct {
		input []byte
		want  int
	}{
		"basic": {
			input: []byte(`{"email": "tim@test.com", "password": "password", "name": "tim"}`),
			want:  201,
		},
		"bad email": {
			input: []byte(`{"email": "not an email", "password": "pass", "name": "a name"}`),
			want:  400,
		},
		"sql injection": {
			input: []byte(`{"email": "'--1=1", "password": "'--1=1", "name": "a name"}`),
			want:  400,
		},
		"missing email field": {
			input: []byte(`{"password": "password", "name": "tim"}`),
			want:  400,
		},
		"empty values": {
			input: []byte(`{"email": "", "password": "", "name": ""}`),
			want:  400,
		},
		"malformed json syntax": {
			input: []byte(`{"email": "tim2@test.com", "password": "pass"`),
			want:  400,
		},
		"wrong data types": {
			input: []byte(`{"email": "tim3@test.com", "password": 12345, "name": "tim"}`),
			want:  400,
		},
		"unexpected fields": {
			input: []byte(`{"email": "tim4@test.com", "password": "password", "name": "tim", "role": "admin"}`),
			want:  400,
		},
		"password too short": {
			input: []byte(`{"email": "tim5@test.com", "password": "123", "name": "tim"}`),
			want:  400,
		},
		"name exceeding max length": {
			input: []byte(`{"email": "tim6@test.com", "password": "password", "name": "this_name_is_way_too_long_and_exceeds_database_column_limits_abcdefghijklmnopqrstuvwxyz"}`),
			want:  400,
		},
		"unicode special characters in name": {
			input: []byte(`{"email": "tim7@test.com", "password": "password", "name": "Tîm 👑"}`),
			want:  201,
		},
		"xss script payload injection": {
			input: []byte(`{"email": "tim8@test.com", "password": "password", "name": "<script>alert(1)</script>"}`),
			want:  201,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(tc.input))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Errorf("Expected status %d, got %d", tc.want, w.Code)
			}
		})
	}

	duplicateCase := []byte(`{"email": "tim@test.com", "password": "different password", "name": "tim"}`)

	t.Run("duplicate email", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(duplicateCase))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestUserLogin(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("Failed to create user for test")
	}

	tests := map[string]struct {
		input []byte
		want  int
	}{
		"basic": {
			input: []byte(`{"email": "tim@test.com", "password": "password"}`),
			want:  200,
		},
		"bad password": {
			input: []byte(`{"email": "tim@test.com", "password": "passwrd"}`),
			want:  401,
		},
		"missing password field": {
			input: []byte(`{"email": "tim@test.com"}`),
			want:  400,
		},
		"empty input values": {
			input: []byte(`{"email": "", "password": ""}`),
			want:  400,
		},
		"malformed json syntax": {
			input: []byte(`{"email": "tim@test.com", "password": "pass"`),
			want:  400,
		},
		"invalid data types": {
			input: []byte(`{"email": "tim@test.com", "password": true}`),
			want:  400,
		},
		"user does not exist": {
			input: []byte(`{"email": "nobody@test.com", "password": "password"}`),
			want:  401,
		},

		// Security Edge Cases
		"sql injection in email": {
			input: []byte(`{"email": "tim@test.com' OR '1'='1", "password": "password"}`),
			want:  401,
		},
		"sql injection in password": {
			input: []byte(`{"email": "tim@test.com", "password": "' OR '1'='1"}`),
			want:  401,
		},
		"case sensitivity check in email": {
			input: []byte(`{"email": "TIM@TEST.COM", "password": "password"}`),
			want:  200,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(tc.input))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Errorf("Expected status %d, got %d", tc.want, w.Code)
			}

			if w.Code >= 200 && w.Code < 300 {
				var response vm.SessionViewModel
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {

				}

				if response.JWT == "" || response.RT == "" {
					t.Errorf("expected token in response body. got: jwt-%s -- rt-%s", response.JWT, response.RT)
				}
			}
		})
	}
}

func TestUserSession(t *testing.T) {
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

	// Create user
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("Failed to create user for test")
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

	// Test private user view
	t.Run("private user view", func(t *testing.T) {
		url := "/api/users/" + user.ID.String()
		req = httptest.NewRequest("GET", url, nil)
		req.AddCookie(jwt)
		req.AddCookie(rt)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to get test user: got status %d", w.Code)
			return
		}

		response = w.Result()
		defer response.Body.Close()

		var privateUser vm.PrivateUserViewModel
		if err := json.NewDecoder(response.Body).Decode(&privateUser); err != nil {
			t.Errorf("failed to decode JSON response: %v", err)
			return
		}

		if privateUser.Email != user.Email {
			t.Errorf("expected user email in response. got: %s - expected: %s", privateUser.Email, user.Email)
		}
	})

	// Test public user view
	t.Run("public user view", func(t *testing.T) {
		url := "/api/users/" + user.ID.String()
		req = httptest.NewRequest("GET", url, nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("Failed to get test user: got status %d", w.Code)
			return
		}

		response = w.Result()
		defer response.Body.Close()

		var publicUser vm.PublicUserViewModel
		decoder := json.NewDecoder(response.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&publicUser); err != nil {
			t.Errorf("Security Leak: Failed to decode public response (possibly leaked private fields): %v", err)
		}
	})
}

func TestUserUpdate(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(testUser.input))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("Failed to create user for test")
	}

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

	// update user
	update := []byte(`{"name":"tim 2"}`)
	url := "/api/users/" + user.ID.String()
	req = httptest.NewRequest("PUT", url, bytes.NewBuffer(update))
	w = httptest.NewRecorder()
	req.AddCookie(jwt)
	req.AddCookie(rt)
	router.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("expected status 204, got: %d", w.Code)
	}

	// get user and verify changes
	userRow, err := cfg.DB.GetUserByEmail(context.Background(), "tim@test.com")
	if err != nil {
		t.Fatalf("couldnt get test user from db: %v", err)
	}

	if userRow.Name != "tim 2" {
		t.Fatalf("user name did not update correctly")
	}
}

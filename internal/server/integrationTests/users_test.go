package server_test

import (
  "bytes"
  "testing"
  "net/http/httptest"

  "github.com/trhys/Recipe-Repo-2/internal/server"
)

func TestCreateUser(t *testing.T) {
  cfg := server.GetConfig()
  tx, err := cfg.DBConn.Begin()
  if err != nil {
    t.Fatalf("Failed to start transaction: %v", err)
  }
  defer tx.Rollback()

  cfg.DB = cfg.DB.WithTx(tx)

  router := server.GetRouter(cfg)

  tests := map[string]struct {
    input  []byte
    want   int
  }{
    "basic": { input: []byte(`{"email": "tim@test.com", "password": "password", "name": "tim"}`), want: 201 },
    "bad email": { input: []byte(`{"email": "not an email", "password": "pass", "name": "a name"}`), want: 400 },
    "sql injection": { input: []byte(`{"email": "'--1=1", "password": "'--1=1", "name": "a name"}`), want: 400 },
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

  router := server.GetRouter(cfg)

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
    input  []byte
    want   int
  }{
    "basic": { input: []byte(`{"email": "tim@test.com", "password": "password"}`), want: 200 },
    "bad password": { input: []byte(`{"email": "tim@test.com", "password": "passwrd"}`), want: 401 },
  }
    
  for name, tc := range tests {
    t.Run(name, func(t *testing.T) {
      req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(tc.input))
	    w := httptest.NewRecorder()
	    router.ServeHTTP(w, req)
    	if w.Code != tc.want {
    		t.Errorf("Expected status %d, got %d", tc.want, w.Code)
    	}
    })
  }
}

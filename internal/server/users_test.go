package server

import (
  "testing"
  "net/http/httptest"
)

func TestCreateUser(t *testing.T) {
  cfg := GetConfig()
  tx, err := cfg.dbConn.Begin()
  if err != nil {
    t.Fatalf("Failed to start transaction: %v", err)
  }
  defer tx.Rollback()

  cfg.db = cfg.db.WithTx(tx)

  router := GetRouter(cfg)

  tests := map[string]struct {
    input  []byte
    want   int
  }{
    "basic": { input: byte[]('{"email": "tim@test.com", "password": "password", "username": "tim"}'), want: 201 },
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
}
    

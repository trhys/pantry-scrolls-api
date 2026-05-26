package server_test

import (
  "log"
  "os"
  "testing"
  "github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
  if err := godotenv.Load("../../../.env"); err != nil {
    log.Fatal("ENV not found")
  }

  os.Exit(m.Run())
}

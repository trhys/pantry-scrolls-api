package server

import (
    "fmt"
    "net/http"

    "github.com/trhys/Recipe-Repo-2/internal/database"
    util "github.com/trhys/Recipe-Repo-2/internal/utility"

func (cfg *ApiConfig) handlerAddMessage(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email string `json:"email"`
        Message string `json:"message"`
    }

    if err := util.DecodeRequest(w, r, 1<<10, &req); err != nil {
        respondFail(w, 400, "Bad request", fmt.Errorf("Failed to decode request: %v", err))
        return
    }

    if err := cfg.DB.AddMessage(r.Context(), database.AddMessageParams{
        Email: req.Email,
        Message: req.Message,
    }); err != nil {
        respondFail(w, 500, "Something went wrong", fmt.Errorf("Query failed: %v", err))
        return
    }

    respondJSON(w, 204, nil)
}
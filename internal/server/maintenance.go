package server

import (
	"fmt"
	"net/http"

    "github.com/google/uuid"
    util "github.com/trhys/Recipe-Repo-2/internal/utility"
    "github.com/trhys/Recipe-Repo-2/internal/database"
)

func (cfg *ApiConfig) handlerGetMaintenance(w http.ResponseWriter, r *http.Request) {
	state, err := cfg.DB.GetServerState(r.Context())
	if err != nil {
		respondFail(r, w, 500, "Failed to retrieve maintenance state", fmt.Errorf("Query failed: %v", err))
		return
	}

	respondJSON(w, 200, struct {
		Active  bool   `json:"active"`
		Message string `json:"message"`
	}{
		Active:  state.Active,
		Message: state.Message,
	})
}

func (cfg *ApiConfig) handlerSetMaintenance(w http.ResponseWriter, r *http.Request) {
  var req struct {
    Active  bool    `json:"active"`
    Message string  `json:"message"`
  }

  // auth
  requesterID, ok := r.Context().Value("userID").(uuid.UUID)
  if !ok {
    respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Invalid uuid at /admin"))
    return
  }

  isAdmin, err := cfg.DB.CheckAdmin(r.Context(), requesterID)
  if !isAdmin || err != nil {
    respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Illegal attempt at /admin: error: %v", err))
    return
  }

  if err := util.DecodeRequest(w, r, 1<<20, &req); err != nil {
    respondFail(r, w, 400, "Bad request", fmt.Errorf("Failed to decode request - ERROR: %v", err))
    return
  }

  if err := cfg.DB.SetServerState(r.Context(), database.SetServerStateParams{
    Active: req.Active,
    Message: req.Message,
  }); err != nil {
    respondFail(r, w, 500, "something went wrong", fmt.Errorf("failed to set maintenance status: %v", err))
    return
  }

  respondJSON(w, 204, nil)
}

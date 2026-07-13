package server

import (
	"fmt"
	"net/http"
)

func (cfg *ApiConfig) handlerGetMaintenance(w http.ResponseWriter, r *http.Request) {
	state, err := cfg.DB.GetServerState(r.Context())
	if err != nil {
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Query failed: %v", err))
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

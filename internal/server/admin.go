package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *ApiConfig) handlerAdminCheck(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Invalid uuid at /admin/check"))
		return
	}

	isAdmin, err := cfg.DB.CheckAdmin(r.Context(), requesterID)
	if err != nil {
		respondFail(r, w, 500, "Something went wrong", err)
		return
	}
	if !isAdmin {
		respondFail(r, w, 403, "Forbidden", fmt.Errorf("User is not an admin"))
		return
	}

	respondJSON(w, 200, nil)
}

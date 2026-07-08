package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	util "github.com/trhys/Recipe-Repo-2/internal/utility"
)

func (cfg *ApiConfig) handlerAddMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		Message string `json:"message"`
	}

	if err := util.DecodeRequest(w, r, 1<<10, &req); err != nil {
		respondFail(r, w, 400, "Bad request", fmt.Errorf("Failed to decode request: %v", err))
		return
	}

	if err := cfg.DB.AddMessage(r.Context(), database.AddMessageParams{
		UserEmail: req.Email,
		Message:   req.Message,
	}); err != nil {
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Query failed: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

func (cfg *ApiConfig) handlerGetMessages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	tag := query.Get("tag")
	if tag == "" {
		tag = "none"
	}

	if tag == "none" {
		messages, err := cfg.DB.GetAllMessages(r.Context())
		if err != nil {
			respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Query failed: %v", err))
			return
		}
		respondJSON(w, 200, messages)
		return
	}

	messages, err := cfg.DB.GetMessagesWithTag(r.Context(), tag)
	if err != nil {
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Query failed: %v", err))
		return
	}

	respondJSON(w, 200, messages)
}

func (cfg *ApiConfig) handlerChangeMessageStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}

	requested := r.PathValue("message_id")
	messageId, err := uuid.Parse(requested)
	if err != nil {
		respondFail(r, w, 404, "Invalid message id", fmt.Errorf("Failed to parse UUID %s : ERROR: %v", requested, err))
		return
	}

	if err := cfg.DB.SetMessageStatus(r.Context(), database.SetMessageStatusParams{
		ID:     messageId,
		Status: req.Status,
	}); err != nil {
		respondFail(r, w, 500, "Something went wrong", fmt.Errorf("Query failed: %v", err))
		return
	}

	respondJSON(w, 204, nil)
}

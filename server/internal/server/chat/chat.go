// Package chat exposes the chat session HTTP endpoints. The coach message
// endpoint (SSE streaming) is layered on top in chunk 5.
package chat

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/server/httpx"
	chatsvc "github.com/radosny/vcoach/server/internal/service/chat"
)

type handler struct {
	svc *chatsvc.Service
}

// Routes registers chat session endpoints on the /api router.
func Routes(r chi.Router, svc *chatsvc.Service) {
	h := &handler{svc: svc}
	r.Get("/chat-sessions", h.list)
	r.Post("/chat-sessions", h.create)
	r.Get("/chat-sessions/{id}/messages", h.messages)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.svc.ListSessions(r.Context(), httpx.AthleteID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if sessions == nil {
		sessions = []entities.ChatSession{}
	}
	httpx.WriteJSON(w, http.StatusOK, sessions)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
			return
		}
	}

	session, err := h.svc.CreateSession(r.Context(), httpx.AthleteID(r.Context()), body.Title)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, session)
}

func (h *handler) messages(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid chat session id %q", chi.URLParam(r, "id")))
		return
	}

	messages, err := h.svc.ListMessages(r.Context(), httpx.AthleteID(r.Context()), sessionID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if messages == nil {
		messages = []entities.ChatMessage{}
	}
	httpx.WriteJSON(w, http.StatusOK, messages)
}

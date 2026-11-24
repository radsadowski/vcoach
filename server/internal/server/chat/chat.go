// Package chat exposes the chat HTTP endpoints, including the SSE coach
// message stream.
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
	coachsvc "github.com/radosny/vcoach/server/internal/service/coach"
)

type handler struct {
	svc   *chatsvc.Service
	coach *coachsvc.Service
}

// Routes registers chat endpoints on the /api router.
func Routes(r chi.Router, svc *chatsvc.Service, coach *coachsvc.Service) {
	h := &handler{svc: svc, coach: coach}
	r.Get("/chat-sessions", h.list)
	r.Post("/chat-sessions", h.create)
	r.Get("/chat-sessions/{id}/messages", h.messages)
	r.Post("/chat-sessions/{id}/messages", h.sendMessage)
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

// sendMessage streams the coach reply as Server-Sent Events. Events:
//
//	event: delta  data: {"delta": "..."}     — final-answer tokens
//	event: done   data: {"messageId": "..."} — reply persisted
//	event: error  data: {"error": "..."}     — the turn failed
func (h *handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid chat session id %q", chi.URLParam(r, "id")))
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}
	if body.Content == "" {
		httpx.WriteError(w, domainerrors.NewValidationError("content is required"))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, domainerrors.NewValidationError("streaming unsupported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	writeEvent := func(event string, data any) bool {
		payload, err := json.Marshal(data)
		if err != nil {
			return false
		}
		if _, err := w.Write([]byte("event: " + event + "\ndata: " + string(payload) + "\n\n")); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	message, err := h.coach.SendMessage(r.Context(), httpx.AthleteID(r.Context()), sessionID, body.Content, func(delta string) {
		writeEvent("delta", map[string]string{"delta": delta})
	})
	if err != nil {
		writeEvent("error", map[string]string{"error": "coach is unavailable: " + err.Error()})
		return
	}

	writeEvent("done", map[string]string{"messageId": message.ID.String()})
}

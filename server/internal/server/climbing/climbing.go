// Package climbing exposes the manual climbing log HTTP endpoints.
package climbing

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/server/httpx"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
)

type handler struct {
	svc *climbingsvc.Service
}

// Routes registers climbing log endpoints on the /api router.
func Routes(r chi.Router, svc *climbingsvc.Service) {
	h := &handler{svc: svc}
	r.Get("/climbing-sessions", h.list)
	r.Post("/climbing-sessions", h.create)
	r.Get("/climbing-sessions/{id}", h.get)
	r.Put("/climbing-sessions/{id}", h.update)
	r.Delete("/climbing-sessions/{id}", h.remove)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	var from, to *string
	if value := r.URL.Query().Get("from"); value != "" {
		from = &value
	}
	if value := r.URL.Query().Get("to"); value != "" {
		to = &value
	}

	sessions, err := h.svc.List(r.Context(), httpx.AthleteID(r.Context()), from, to)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if sessions == nil {
		sessions = []entities.ClimbingSession{}
	}
	httpx.WriteJSON(w, http.StatusOK, sessions)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var input entities.CreateClimbingSessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}

	session, err := h.svc.Create(r.Context(), httpx.AthleteID(r.Context()), input)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, session)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	session, err := h.svc.Get(r.Context(), httpx.AthleteID(r.Context()), parseID(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	var input entities.UpdateClimbingSessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}

	session, err := h.svc.Update(r.Context(), httpx.AthleteID(r.Context()), parseID(r, "id"), input)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), httpx.AthleteID(r.Context()), parseID(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request, param string) uuid.UUID {
	// Handlers registered only with valid UUID patterns call this after
	// routing; a parse failure here means the route pattern was bypassed.
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		return uuid.Nil
	}
	return id
}

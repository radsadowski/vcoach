// Package goals exposes the goal HTTP endpoints.
package goals

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/server/httpx"
	goalsvc "github.com/radosny/vcoach/server/internal/service/goals"
)

type handler struct {
	svc *goalsvc.Service
}

// Routes registers goal endpoints on the /api router.
func Routes(r chi.Router, svc *goalsvc.Service) {
	h := &handler{svc: svc}
	r.Get("/goals", h.list)
	r.Post("/goals", h.create)
	r.Patch("/goals/{id}", h.update)
	r.Delete("/goals/{id}", h.remove)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	goals, err := h.svc.ListGoals(r.Context(), httpx.AthleteID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if goals == nil {
		goals = []entities.Goal{}
	}
	httpx.WriteJSON(w, http.StatusOK, goals)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var input entities.CreateGoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}

	goal, err := h.svc.CreateGoal(r.Context(), httpx.AthleteID(r.Context()), input)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, goal)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	goalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid goal id %q", chi.URLParam(r, "id")))
		return
	}

	var input entities.UpdateGoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}

	goal, err := h.svc.UpdateGoal(r.Context(), httpx.AthleteID(r.Context()), goalID, input)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, goal)
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	goalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid goal id %q", chi.URLParam(r, "id")))
		return
	}

	if err := h.svc.DeleteGoal(r.Context(), httpx.AthleteID(r.Context()), goalID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

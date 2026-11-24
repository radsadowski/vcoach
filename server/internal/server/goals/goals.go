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
	r.Get("/goals/{id}/progress", h.progress)
	r.Patch("/goals/{id}", h.update)
	r.Delete("/goals/{id}", h.remove)
}

// progress godoc
// @Summary      Get goal progress
// @Description  Progress facts for one goal: run goals (longest run 30d, best pace, weekly volumes) or climb goals (hardest sent 30d, rank gap to target, attempts).
// @Tags         goals
// @Produce      json
// @Param        id path string true "Goal ID"
// @Success      200 {object} goals.Progress
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /goals/{id}/progress [get]
func (h *handler) progress(w http.ResponseWriter, r *http.Request) {
	goalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid goal id %q", chi.URLParam(r, "id")))
		return
	}

	progress, err := h.svc.GetProgress(r.Context(), httpx.AthleteID(r.Context()), goalID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, progress)
}

// list godoc
// @Summary      List goals
// @Description  All goals of the athlete, newest first.
// @Tags         goals
// @Produce      json
// @Success      200 {array} entities.Goal
// @Router       /goals [get]
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

// create godoc
// @Summary      Create a goal
// @Description  Catalog-validated goal (run events in km, boulder and lead goals as Font grades). Setting isPrimary clears other active goals.
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        body body entities.CreateGoalInput true "Goal to create"
// @Success      201 {object} entities.Goal
// @Failure      400 {object} httpx.ErrorResponse
// @Router       /goals [post]
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

// update godoc
// @Summary      Update a goal
// @Description  Partial update of status, primary flag and notes. Setting isPrimary clears other active goals.
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        id   path string                 true "Goal ID"
// @Param        body body entities.UpdateGoalInput true "Fields to update"
// @Success      200 {object} entities.Goal
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /goals/{id} [patch]
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

// remove godoc
// @Summary      Delete a goal
// @Tags         goals
// @Param        id path string true "Goal ID"
// @Success      204 "No content"
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /goals/{id} [delete]
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

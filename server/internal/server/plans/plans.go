// Package plans exposes the training plan HTTP endpoints.
package plans

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/server/httpx"
	plansvc "github.com/radosny/vcoach/server/internal/service/plans"
)

type handler struct {
	svc *plansvc.Service
}

// Routes registers plan endpoints on the /api router.
func Routes(r chi.Router, svc *plansvc.Service) {
	h := &handler{svc: svc}
	r.Get("/plans", h.list)
	r.Post("/plans", h.create)
	r.Get("/plans/{id}", h.get)
	r.Delete("/plans/{id}", h.archive)
}

// list godoc
// @Summary      List plan versions
// @Description  All plan versions of the athlete, newest version first.
// @Tags         plans
// @Produce      json
// @Success      200 {array} entities.Plan
// @Router       /plans [get]
func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context(), httpx.AthleteID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if plans == nil {
		plans = []entities.Plan{}
	}
	httpx.WriteJSON(w, http.StatusOK, plans)
}

// create godoc
// @Summary      Create a training plan
// @Description  Creates the next plan version and archives the current active one atomically. Weeks need consecutive weekNumbers starting at 1; sessions carry YYYY-MM-DD planned dates.
// @Tags         plans
// @Accept       json
// @Produce      json
// @Param        body body entities.CreatePlanInput true "Plan to create"
// @Success      201 {object} entities.PlanWithSessions
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /plans [post]
func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var input entities.CreatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}

	plan, err := h.svc.CreatePlan(r.Context(), httpx.AthleteID(r.Context()), input)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, plan)
}

// get godoc
// @Summary      Get a plan
// @Description  One plan version with weeks, sessions and per-session compliance.
// @Tags         plans
// @Produce      json
// @Param        id path string true "Plan ID"
// @Success      200 {object} entities.PlanWithSessions
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /plans/{id} [get]
func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	planID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid plan id %q", chi.URLParam(r, "id")))
		return
	}

	plan, err := h.svc.GetPlan(r.Context(), httpx.AthleteID(r.Context()), planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, plan)
}

// archive godoc
// @Summary      Archive the active plan
// @Description  Archives a plan version (e.g. before regenerating via chat).
// @Tags         plans
// @Param        id path string true "Plan ID"
// @Success      204 "No content"
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /plans/{id} [delete]
func (h *handler) archive(w http.ResponseWriter, r *http.Request) {
	planID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid plan id %q", chi.URLParam(r, "id")))
		return
	}

	if err := h.svc.ArchivePlan(r.Context(), httpx.AthleteID(r.Context()), planID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

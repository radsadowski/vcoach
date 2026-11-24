// Package profile exposes the athlete profile HTTP endpoints.
package profile

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/server/httpx"
	profilesvc "github.com/radosny/vcoach/server/internal/service/profile"
)

type handler struct {
	svc *profilesvc.Service
}

// Routes registers profile endpoints on the /api router.
func Routes(r chi.Router, svc *profilesvc.Service) {
	h := &handler{svc: svc}
	r.Get("/profile", h.get)
	r.Put("/profile", h.update)
}

// get godoc
// @Summary      Get the athlete profile
// @Description  Manual basics plus metrics derived from synced Garmin data (HR zones, VO2max, weekly volumes, longest run, weight trend).
// @Tags         profile
// @Produce      json
// @Success      200 {object} profile.AthleteWithDerived
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /profile [get]
func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	athlete, err := h.svc.GetWithDerived(r.Context(), httpx.AthleteID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, athlete)
}

// update godoc
// @Summary      Update profile basics
// @Description  Updates manual profile basics; nil fields are kept.
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        body body entities.UpdateAthleteInput true "Profile basics"
// @Success      200 {object} profile.AthleteWithDerived
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /profile [put]
func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	var input entities.UpdateAthleteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.WriteError(w, domainerrors.NewValidationError("invalid JSON body: %v", err))
		return
	}

	athlete, err := h.svc.UpdateBasics(r.Context(), httpx.AthleteID(r.Context()), input)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, athlete)
}

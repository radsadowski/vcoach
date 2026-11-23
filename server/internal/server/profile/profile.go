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

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	athlete, err := h.svc.GetWithDerived(r.Context(), httpx.AthleteID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, athlete)
}

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

// Package sync exposes the Garmin sync HTTP endpoints.
package sync

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/radosny/vcoach/server/internal/server/httpx"
	syncsvc "github.com/radosny/vcoach/server/internal/service/sync"
)

type handler struct {
	svc *syncsvc.Service
}

// Routes registers sync endpoints on the /api router.
func Routes(r chi.Router, svc *syncsvc.Service) {
	h := &handler{svc: svc}
	r.Post("/sync", h.syncNow)
	r.Get("/sync/status", h.status)
}

func (h *handler) syncNow(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	result, err := h.svc.SyncNow(ctx, httpx.AthleteID(r.Context()))
	if err != nil && result.ActivitiesSynced == 0 && result.HealthDaysSynced == 0 && len(result.Errors) == 0 {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.Status(r.Context(), httpx.AthleteID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, status)
}

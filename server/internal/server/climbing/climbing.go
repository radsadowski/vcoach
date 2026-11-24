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

// list godoc
// @Summary      List climbing sessions
// @Description  Manual climbing log entries in the inclusive date range.
// @Tags         climbing
// @Produce      json
// @Param        from query string false "Range start (YYYY-MM-DD)"
// @Param        to   query string false "Range end (YYYY-MM-DD)"
// @Success      200 {array} entities.ClimbingSession
// @Failure      400 {object} httpx.ErrorResponse
// @Router       /climbing-sessions [get]
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

// create godoc
// @Summary      Log a climbing session
// @Description  Boulder and lead grades use the Font scale (e.g. 6B, 7A boulder; 6a, 7a+ lead). Hardest sent cannot be harder than hardest attempted.
// @Tags         climbing
// @Accept       json
// @Produce      json
// @Param        body body entities.CreateClimbingSessionInput true "Session to log"
// @Success      201 {object} entities.ClimbingSession
// @Failure      400 {object} httpx.ErrorResponse
// @Router       /climbing-sessions [post]
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

// get godoc
// @Summary      Get a climbing session
// @Tags         climbing
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} entities.ClimbingSession
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /climbing-sessions/{id} [get]
func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	session, err := h.svc.Get(r.Context(), httpx.AthleteID(r.Context()), parseID(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

// update godoc
// @Summary      Update a climbing session
// @Description  Partial update; grades are re-validated after merging with the stored session.
// @Tags         climbing
// @Accept       json
// @Produce      json
// @Param        id   path string                            true "Session ID"
// @Param        body body entities.UpdateClimbingSessionInput true "Fields to update"
// @Success      200 {object} entities.ClimbingSession
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /climbing-sessions/{id} [put]
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

// remove godoc
// @Summary      Delete a climbing session
// @Tags         climbing
// @Param        id path string true "Session ID"
// @Success      204 "No content"
// @Failure      400 {object} httpx.ErrorResponse
// @Failure      404 {object} httpx.ErrorResponse
// @Router       /climbing-sessions/{id} [delete]
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

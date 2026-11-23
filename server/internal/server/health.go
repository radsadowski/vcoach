package server

import (
	"net/http"

	"github.com/radosny/vcoach/server/internal/server/httpx"
)

// handleHealthz is the infrastructure health endpoint: no DB, no auth.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handlePing proves the full /api pipeline: athlete resolver + handler.
func handlePing(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"pong": true, "athlete_id": httpx.AthleteID(r.Context())})
}

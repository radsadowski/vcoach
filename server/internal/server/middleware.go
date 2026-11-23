package server

import (
	"net/http"

	"github.com/radosny/vcoach/server/internal/server/httpx"
)

// resolveAthlete resolves the current athlete for every /api request and puts
// it into the request context.
func (d *Deps) resolveAthlete(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		athleteID, err := d.Athletes.GetOrCreateDefaultAthlete(r.Context())
		if err != nil {
			d.Log.ErrorContext(r.Context(), "resolve athlete", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r.WithContext(httpx.WithAthleteID(r.Context(), athleteID)))
	})
}

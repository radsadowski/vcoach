// Package server is the HTTP layer: request/response translation only.
// Business logic lives in internal/service.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/radosny/vcoach/server/internal/server/chat"
	"github.com/radosny/vcoach/server/internal/server/climbing"
	"github.com/radosny/vcoach/server/internal/server/goals"
	"github.com/radosny/vcoach/server/internal/server/plans"
	"github.com/radosny/vcoach/server/internal/server/profile"
	syncapi "github.com/radosny/vcoach/server/internal/server/sync"
	chatsvc "github.com/radosny/vcoach/server/internal/service/chat"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
	coachsvc "github.com/radosny/vcoach/server/internal/service/coach"
	goalsvc "github.com/radosny/vcoach/server/internal/service/goals"
	plansvc "github.com/radosny/vcoach/server/internal/service/plans"
	profilesvc "github.com/radosny/vcoach/server/internal/service/profile"
	syncsvc "github.com/radosny/vcoach/server/internal/service/sync"
)

// Deps carries the router's collaborators.
type Deps struct {
	Log      *slog.Logger
	Athletes AthleteStore
	Profile  *profilesvc.Service
	Goals    *goalsvc.Service
	Plans    *plansvc.Service
	Climbing *climbingsvc.Service
	Chat     *chatsvc.Service
	Sync     *syncsvc.Service
	Coach    *coachsvc.Service
}

// AthleteStore resolves the current athlete. Implemented by the repository;
// kept as an interface so handlers stay testable without a database.
type AthleteStore interface {
	GetOrCreateDefaultAthlete(ctx context.Context) (uuid.UUID, error)
}

// NewRouter builds the HTTP router.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware(deps.Log))

	r.Get("/healthz", handleHealthz)

	// Swagger UI: schema browsing and direct requests against the API.
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api", func(api chi.Router) {
		api.Use(deps.resolveAthlete)
		api.Get("/ping", handlePing)

		profile.Routes(api, deps.Profile)
		goals.Routes(api, deps.Goals)
		plans.Routes(api, deps.Plans)
		climbing.Routes(api, deps.Climbing)
		chat.Routes(api, deps.Chat, deps.Coach)
		syncapi.Routes(api, deps.Sync)
	})

	return r
}

func loggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				log.Info("http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", rec.status,
					"duration", time.Since(start).Round(time.Millisecond),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// Flush passes SSE flushes through the access-log recorder.
func (rec *statusRecorder) Flush() {
	if flusher, ok := rec.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

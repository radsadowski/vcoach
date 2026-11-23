package entities

import (
	"time"

	"github.com/google/uuid"
)

// PlanStatus is the lifecycle state of a plan version.
type PlanStatus string

const (
	PlanStatusActive   PlanStatus = "active"
	PlanStatusArchived PlanStatus = "archived"
)

// SessionStatus is the compliance state of a planned session.
type SessionStatus string

const (
	SessionStatusPlanned   SessionStatus = "planned"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusMissed    SessionStatus = "missed"
	SessionStatusSkipped   SessionStatus = "skipped"
)

// Sport families used for compliance matching and goal progress.
const (
	SportRun      = "run"
	SportBoulder  = "boulder"
	SportLead     = "lead"
	SportStrength = "strength"
	SportOther    = "other"
)

// Plan is one version of an athlete's training plan. Plans are append-only:
// adapting creates a new version and archives its predecessor.
type Plan struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	AthleteID uuid.UUID  `db:"athlete_id" json:"athleteId"`
	GoalID    *uuid.UUID `db:"goal_id" json:"goalId,omitempty"`
	Version   int        `db:"version" json:"version"`
	Status    PlanStatus `db:"status" json:"status"`
	Title     string     `db:"title" json:"title"`
	Notes     string     `db:"notes" json:"notes"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
}

// PlanWeek is one training week inside a plan.
type PlanWeek struct {
	ID         uuid.UUID `db:"id" json:"id"`
	PlanID     uuid.UUID `db:"plan_id" json:"planId"`
	WeekNumber int       `db:"week_number" json:"weekNumber"`
	StartDate  Date      `db:"start_date" json:"startDate"`
	Focus      string    `db:"focus" json:"focus"`
}

// PlanSession is a single planned session; Garmin activities get matched
// against it for compliance.
type PlanSession struct {
	ID                uuid.UUID     `db:"id" json:"id"`
	WeekID            uuid.UUID     `db:"week_id" json:"weekId"`
	AthleteID         uuid.UUID     `db:"athlete_id" json:"athleteId"`
	Sport             string        `db:"sport" json:"sport"`
	SessionType       string        `db:"session_type" json:"sessionType"`
	Description       string        `db:"description" json:"description"`
	PlannedDate       Date          `db:"planned_date" json:"plannedDate"`
	TargetDurationMin *int          `db:"target_duration_min" json:"targetDurationMin,omitempty"`
	TargetDistanceKm  *float64      `db:"target_distance_km" json:"targetDistanceKm,omitempty"`
	TargetIntensity   *string       `db:"target_intensity" json:"targetIntensity,omitempty"`
	Status            SessionStatus `db:"status" json:"status"`
	MatchedActivityID *uuid.UUID    `db:"matched_activity_id" json:"matchedActivityId,omitempty"`
	CompletedAt       *time.Time    `db:"completed_at" json:"completedAt,omitempty"`
}

// PlanWithSessions is the full plan tree as served by the API.
type PlanWithSessions struct {
	Plan
	Weeks []PlanWeekWithSessions `json:"weeks"`
}

// PlanWeekWithSessions is a week with its sessions.
type PlanWeekWithSessions struct {
	PlanWeek
	Sessions []PlanSession `json:"sessions"`
}

// CreatePlanInput is the validated payload for creating a plan version.
type CreatePlanInput struct {
	GoalID *uuid.UUID        `json:"goalId,omitempty"`
	Title  string            `json:"title"`
	Notes  string            `json:"notes"`
	Weeks  []CreateWeekInput `json:"weeks"`
}

// CreateWeekInput is one week in a plan creation payload.
type CreateWeekInput struct {
	WeekNumber int                  `json:"weekNumber"`
	StartDate  Date                 `json:"startDate"`
	Focus      string               `json:"focus"`
	Sessions   []CreateSessionInput `json:"sessions"`
}

// CreateSessionInput is one session in a plan creation payload.
type CreateSessionInput struct {
	Sport             string   `json:"sport"`
	SessionType       string   `json:"sessionType"`
	Description       string   `json:"description"`
	PlannedDate       Date     `json:"plannedDate"`
	TargetDurationMin *int     `json:"targetDurationMin,omitempty"`
	TargetDistanceKm  *float64 `json:"targetDistanceKm,omitempty"`
	TargetIntensity   *string  `json:"targetIntensity,omitempty"`
}

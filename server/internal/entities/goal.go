package entities

import (
	"time"

	"github.com/google/uuid"
)

// GoalActivityType is the activity family a goal targets.
type GoalActivityType string

const (
	GoalActivityRun     GoalActivityType = "run"
	GoalActivityBoulder GoalActivityType = "boulder"
	GoalActivityLead    GoalActivityType = "lead"
)

// GoalStatus is the lifecycle state of a goal.
type GoalStatus string

const (
	GoalStatusActive    GoalStatus = "active"
	GoalStatusAchieved  GoalStatus = "achieved"
	GoalStatusAbandoned GoalStatus = "abandoned"
)

// Target metrics used by the goal catalog.
const (
	TargetMetricDistanceKm = "distance_km"
	TargetMetricGrade      = "grade"
)

// Goal is a structured, catalog-typed training goal. Multiple goals can be
// active, exactly one of them primary; plans are built around the primary.
type Goal struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	AthleteID    uuid.UUID        `db:"athlete_id" json:"athleteId"`
	ActivityType GoalActivityType `db:"activity_type" json:"activityType"`
	TargetMetric string           `db:"target_metric" json:"targetMetric"`
	TargetValue  string           `db:"target_value" json:"targetValue"`
	TargetDate   *Date            `db:"target_date" json:"targetDate,omitempty"`
	IsPrimary    bool             `db:"is_primary" json:"isPrimary"`
	Status       GoalStatus       `db:"status" json:"status"`
	Notes        string           `db:"notes" json:"notes"`
	CreatedAt    time.Time        `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updatedAt"`
}

// CreateGoalInput is the validated payload for creating a goal.
type CreateGoalInput struct {
	ActivityType GoalActivityType `json:"activityType"`
	TargetMetric string           `json:"targetMetric"`
	TargetValue  string           `json:"targetValue"`
	TargetDate   *Date            `json:"targetDate,omitempty"`
	Notes        string           `json:"notes"`
	IsPrimary    bool             `json:"isPrimary"`
}

// UpdateGoalInput is the validated payload for patching a goal.
type UpdateGoalInput struct {
	Status    *GoalStatus `json:"status,omitempty"`
	IsPrimary *bool       `json:"isPrimary,omitempty"`
	Notes     *string     `json:"notes,omitempty"`
}

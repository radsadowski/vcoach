// Package goal provides functional-options builders for goal test data.
package goal

import (
	"github.com/radosny/vcoach/server/internal/entities"
)

// Option mutates a CreateGoalInput.
type Option func(*entities.CreateGoalInput)

// NewInput returns a valid default goal input (10 km run) with options applied.
func NewInput(opts ...Option) entities.CreateGoalInput {
	input := entities.CreateGoalInput{
		ActivityType: entities.GoalActivityRun,
		TargetMetric: entities.TargetMetricDistanceKm,
		TargetValue:  "10",
	}
	for _, opt := range opts {
		opt(&input)
	}
	return input
}

// WithRunDistance targets a run event of the given distance in km.
func WithRunDistance(km string) Option {
	return func(input *entities.CreateGoalInput) {
		input.ActivityType = entities.GoalActivityRun
		input.TargetMetric = entities.TargetMetricDistanceKm
		input.TargetValue = km
	}
}

// WithBoulderGrade targets a Font boulder grade (e.g. 7C).
func WithBoulderGrade(grade string) Option {
	return func(input *entities.CreateGoalInput) {
		input.ActivityType = entities.GoalActivityBoulder
		input.TargetMetric = entities.TargetMetricGrade
		input.TargetValue = grade
	}
}

// WithLeadGrade targets a Font lead grade (e.g. 7a).
func WithLeadGrade(grade string) Option {
	return func(input *entities.CreateGoalInput) {
		input.ActivityType = entities.GoalActivityLead
		input.TargetMetric = entities.TargetMetricGrade
		input.TargetValue = grade
	}
}

// WithPrimary marks the goal primary.
func WithPrimary() Option {
	return func(input *entities.CreateGoalInput) {
		input.IsPrimary = true
	}
}

// WithTargetDate sets the goal deadline.
func WithTargetDate(date entities.Date) Option {
	return func(input *entities.CreateGoalInput) {
		input.TargetDate = &date
	}
}

// WithNotes attaches notes to the goal.
func WithNotes(notes string) Option {
	return func(input *entities.CreateGoalInput) {
		input.Notes = notes
	}
}

// WithActivityType overrides the activity type (for invalid-input tests).
func WithActivityType(activityType entities.GoalActivityType) Option {
	return func(input *entities.CreateGoalInput) {
		input.ActivityType = activityType
	}
}

// WithTargetMetric overrides the target metric (for invalid-input tests).
func WithTargetMetric(metric string) Option {
	return func(input *entities.CreateGoalInput) {
		input.TargetMetric = metric
	}
}

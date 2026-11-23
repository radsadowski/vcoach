// Package plan provides functional-options builders for plan test data.
package plan

import (
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

// Option mutates a CreatePlanInput.
type Option func(*entities.CreatePlanInput)

// NewInput returns a valid default plan input (4 weeks, 2 run sessions per
// week starting today) with options applied.
func NewInput(opts ...Option) entities.CreatePlanInput {
	input := entities.CreatePlanInput{
		Title: "Test plan",
		Notes: "built by test builder",
		Weeks: makeWeeks(4, 2),
	}
	for _, opt := range opts {
		opt(&input)
	}
	return input
}

// WithWeekCount sets the number of weeks (2 easy run sessions each).
func WithWeekCount(weeks int) Option {
	return func(input *entities.CreatePlanInput) {
		input.Weeks = makeWeeks(weeks, 2)
	}
}

// WithGoalID ties the plan to a goal.
func WithGoalID(goalID uuid.UUID) Option {
	return func(input *entities.CreatePlanInput) {
		input.GoalID = &goalID
	}
}

// WithTitle sets the plan title.
func WithTitle(title string) Option {
	return func(input *entities.CreatePlanInput) {
		input.Title = title
	}
}

// WithWeeks replaces the weeks entirely.
func WithWeeks(weeks []entities.CreateWeekInput) Option {
	return func(input *entities.CreatePlanInput) {
		input.Weeks = weeks
	}
}

func makeWeeks(count, sessionsPerWeek int) []entities.CreateWeekInput {
	today := time.Now().UTC()
	result := make([]entities.CreateWeekInput, 0, count)
	for week := 0; week < count; week++ {
		weekStart := today.AddDate(0, 0, 7*week)
		sessions := make([]entities.CreateSessionInput, 0, sessionsPerWeek)
		for session := 0; session < sessionsPerWeek; session++ {
			sessions = append(sessions, entities.CreateSessionInput{
				Sport:       entities.SportRun,
				SessionType: "easy",
				Description: "easy run",
				PlannedDate: entities.Date(weekStart.AddDate(0, 0, session*2)),
			})
		}
		result = append(result, entities.CreateWeekInput{
			WeekNumber: week + 1,
			StartDate:  entities.Date(weekStart),
			Focus:      "base",
			Sessions:   sessions,
		})
	}
	return result
}

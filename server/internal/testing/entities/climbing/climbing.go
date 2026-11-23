// Package climbing provides functional-options builders for climbing log
// test data.
package climbing

import (
	"time"

	"github.com/radosny/vcoach/server/internal/entities"
)

// Option mutates a CreateClimbingSessionInput.
type Option func(*entities.CreateClimbingSessionInput)

// NewInput returns a valid default climbing session (boulder, 6A sent, 6B
// attempted, today) with options applied.
func NewInput(opts ...Option) entities.CreateClimbingSessionInput {
	input := entities.CreateClimbingSessionInput{
		SessionDate:      entities.Date(time.Now().UTC()),
		Venue:            "Test gym",
		Discipline:       "boulder",
		HardestSent:      "6A",
		HardestAttempted: "6B",
		Attempts:         12,
		Notes:            "built by test builder",
	}
	for _, opt := range opts {
		opt(&input)
	}
	return input
}

// WithDiscipline sets boulder or lead.
func WithDiscipline(discipline string) Option {
	return func(input *entities.CreateClimbingSessionInput) {
		input.Discipline = discipline
	}
}

// WithSent sets the hardest grade sent.
func WithSent(grade string) Option {
	return func(input *entities.CreateClimbingSessionInput) {
		input.HardestSent = grade
	}
}

// WithAttempted sets the hardest grade attempted.
func WithAttempted(grade string) Option {
	return func(input *entities.CreateClimbingSessionInput) {
		input.HardestAttempted = grade
	}
}

// WithAttempts sets the number of hard attempts.
func WithAttempts(attempts int) Option {
	return func(input *entities.CreateClimbingSessionInput) {
		input.Attempts = attempts
	}
}

// WithRPE sets the perceived exertion (1-10).
func WithRPE(rpe int) Option {
	return func(input *entities.CreateClimbingSessionInput) {
		input.RPE = &rpe
	}
}

// WithDate sets the session date.
func WithDate(date time.Time) Option {
	return func(input *entities.CreateClimbingSessionInput) {
		input.SessionDate = entities.Date(date)
	}
}

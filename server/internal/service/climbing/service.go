// Package climbing implements the manual climbing log business logic.
// Goal progress for boulder/lead goals is computed from these sessions.
package climbing

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
)

// Service holds the climbing log business logic dependencies.
type Service struct {
	repo *repository.Repository
}

// New builds a climbing Service.
func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// Create validates and inserts a climbing session.
func (s *Service) Create(ctx context.Context, athleteID uuid.UUID, input entities.CreateClimbingSessionInput) (entities.ClimbingSession, error) {
	if input.SessionDate.IsZero() {
		return entities.ClimbingSession{}, domainerrors.NewValidationError("sessionDate is required")
	}
	if err := validateSession(input.Discipline, input.HardestSent, input.HardestAttempted, input.Attempts, input.RPE); err != nil {
		return entities.ClimbingSession{}, err
	}
	return s.repo.CreateClimbingSession(ctx, athleteID, input)
}

// List returns climbing sessions in the inclusive YYYY-MM-DD date range.
// Nil bounds are open-ended.
func (s *Service) List(ctx context.Context, athleteID uuid.UUID, from, to *string) ([]entities.ClimbingSession, error) {
	fromDate, err := parseOptionalDate("from", from)
	if err != nil {
		return nil, err
	}
	toDate, err := parseOptionalDate("to", to)
	if err != nil {
		return nil, err
	}
	return s.repo.ListClimbingSessions(ctx, athleteID, fromDate, toDate)
}

// Get returns one climbing session scoped to the athlete.
func (s *Service) Get(ctx context.Context, athleteID, sessionID uuid.UUID) (entities.ClimbingSession, error) {
	session, err := s.repo.GetClimbingSession(ctx, athleteID, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.ClimbingSession{}, domainerrors.NewNotFoundError("climbing session %s not found", sessionID)
		}
		return entities.ClimbingSession{}, err
	}
	return session, nil
}

// Update applies a partial update; grades are re-validated after merging
// with the stored session.
func (s *Service) Update(ctx context.Context, athleteID, sessionID uuid.UUID, input entities.UpdateClimbingSessionInput) (entities.ClimbingSession, error) {
	current, err := s.Get(ctx, athleteID, sessionID)
	if err != nil {
		return entities.ClimbingSession{}, err
	}

	merged := mergeSession(current, input)
	if err := validateSession(merged.discipline, merged.hardestSent, merged.hardestAttempted, merged.attempts, merged.rpe); err != nil {
		return entities.ClimbingSession{}, err
	}

	updated, err := s.repo.UpdateClimbingSession(ctx, athleteID, sessionID, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.ClimbingSession{}, domainerrors.NewNotFoundError("climbing session %s not found", sessionID)
		}
		return entities.ClimbingSession{}, err
	}
	return updated, nil
}

// Delete removes a climbing session.
func (s *Service) Delete(ctx context.Context, athleteID, sessionID uuid.UUID) error {
	if err := s.repo.DeleteClimbingSession(ctx, athleteID, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainerrors.NewNotFoundError("climbing session %s not found", sessionID)
		}
		return err
	}
	return nil
}

func validateSession(discipline, hardestSent, hardestAttempted string, attempts int, rpe *int) error {
	var validGrade func(string) bool
	var rank func(string) (int, error)
	switch discipline {
	case "boulder":
		validGrade = entities.ValidBoulderGrade
		rank = entities.BoulderGradeRank
	case "lead":
		validGrade = entities.ValidLeadGrade
		rank = entities.LeadGradeRank
	default:
		return domainerrors.NewValidationError("discipline must be boulder or lead, got %q", discipline)
	}

	if !validGrade(hardestSent) {
		return domainerrors.NewValidationError("hardestSent must be a valid %s grade (e.g. %s), got %q", discipline, exampleGrade(discipline), hardestSent)
	}
	if !validGrade(hardestAttempted) {
		return domainerrors.NewValidationError("hardestAttempted must be a valid %s grade, got %q", discipline, hardestAttempted)
	}

	sentRank, _ := rank(hardestSent)
	attemptedRank, _ := rank(hardestAttempted)
	if sentRank > attemptedRank {
		return domainerrors.NewValidationError("hardestSent (%s) cannot be harder than hardestAttempted (%s)", hardestSent, hardestAttempted)
	}

	if attempts < 0 {
		return domainerrors.NewValidationError("attempts must be >= 0, got %d", attempts)
	}
	if rpe != nil && (*rpe < 1 || *rpe > 10) {
		return domainerrors.NewValidationError("rpe must be between 1 and 10, got %d", *rpe)
	}
	return nil
}

func exampleGrade(discipline string) string {
	if discipline == "boulder" {
		return "6B, 7A, 7C"
	}
	return "6a, 7a+"
}

func parseOptionalDate(name string, value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, *value)
	if err != nil {
		return nil, domainerrors.NewValidationError("%s must be a YYYY-MM-DD date, got %q", name, *value)
	}
	return &parsed, nil
}

type mergedSession struct {
	discipline       string
	hardestSent      string
	hardestAttempted string
	attempts         int
	rpe              *int
}

func mergeSession(current entities.ClimbingSession, input entities.UpdateClimbingSessionInput) mergedSession {
	merged := mergedSession{
		discipline:       current.Discipline,
		hardestSent:      current.HardestSent,
		hardestAttempted: current.HardestAttempted,
		attempts:         current.Attempts,
		rpe:              current.RPE,
	}
	if input.Discipline != nil {
		merged.discipline = *input.Discipline
	}
	if input.HardestSent != nil {
		merged.hardestSent = *input.HardestSent
	}
	if input.HardestAttempted != nil {
		merged.hardestAttempted = *input.HardestAttempted
	}
	if input.Attempts != nil {
		merged.attempts = *input.Attempts
	}
	if input.RPE != nil {
		merged.rpe = input.RPE
	}
	return merged
}

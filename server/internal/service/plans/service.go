// Package plans implements training plan business logic. Plans are
// append-only: adapting means creating a new version, which archives its
// predecessor atomically.
package plans

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
)

// maxPlanWeeks bounds LLM-generated plans to a sane length.
const maxPlanWeeks = 52

// validPlanSports are the sports a planned session can target. They mirror
// the sport families used for Garmin compliance matching.
var validPlanSports = map[string]struct{}{
	entities.SportRun:      {},
	entities.SportBoulder:  {},
	entities.SportLead:     {},
	entities.SportStrength: {},
	entities.SportOther:    {},
}

// Service holds the plans business logic dependencies.
type Service struct {
	repo *repository.Repository
}

// New builds a plans Service.
func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreatePlan validates input and creates the next plan version, archiving the
// current active plan in the same transaction.
func (s *Service) CreatePlan(ctx context.Context, athleteID uuid.UUID, input entities.CreatePlanInput) (entities.PlanWithSessions, error) {
	if err := validatePlanInput(&input); err != nil {
		return entities.PlanWithSessions{}, err
	}

	if input.GoalID != nil {
		exists, err := s.repo.GoalExists(ctx, athleteID, *input.GoalID)
		if err != nil {
			return entities.PlanWithSessions{}, err
		}
		if !exists {
			return entities.PlanWithSessions{}, domainerrors.NewNotFoundError("goal %s not found", *input.GoalID)
		}
	}

	return s.repo.CreatePlanVersion(ctx, athleteID, input.GoalID, input)
}

// ListPlans returns all plan versions.
func (s *Service) ListPlans(ctx context.Context, athleteID uuid.UUID) ([]entities.Plan, error) {
	return s.repo.ListPlans(ctx, athleteID)
}

// GetPlan returns one plan with weeks and sessions.
func (s *Service) GetPlan(ctx context.Context, athleteID, planID uuid.UUID) (entities.PlanWithSessions, error) {
	plan, err := s.repo.GetPlan(ctx, athleteID, planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.PlanWithSessions{}, domainerrors.NewNotFoundError("plan %s not found", planID)
		}
		return entities.PlanWithSessions{}, err
	}
	return plan, nil
}

// ArchivePlan archives the active plan (e.g. before regenerating via chat).
func (s *Service) ArchivePlan(ctx context.Context, athleteID, planID uuid.UUID) error {
	if err := s.repo.ArchivePlan(ctx, athleteID, planID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainerrors.NewNotFoundError("plan %s not found or not active", planID)
		}
		return err
	}
	return nil
}

func validatePlanInput(input *entities.CreatePlanInput) error {
	if input.Title == "" {
		input.Title = "Training plan"
	}
	if len(input.Weeks) == 0 {
		return domainerrors.NewValidationError("plan must contain at least one week")
	}
	if len(input.Weeks) > maxPlanWeeks {
		return domainerrors.NewValidationError("plan must contain at most %d weeks, got %d", maxPlanWeeks, len(input.Weeks))
	}

	seenWeeks := make(map[int]struct{}, len(input.Weeks))
	for _, week := range input.Weeks {
		if week.WeekNumber < 1 {
			return domainerrors.NewValidationError("weekNumber must be >= 1, got %d", week.WeekNumber)
		}
		if _, ok := seenWeeks[week.WeekNumber]; ok {
			return domainerrors.NewValidationError("weekNumber %d appears more than once", week.WeekNumber)
		}
		seenWeeks[week.WeekNumber] = struct{}{}

		if week.StartDate.IsZero() {
			return domainerrors.NewValidationError("week %d: startDate is required", week.WeekNumber)
		}

		for i, session := range week.Sessions {
			if err := validateSessionInput(week.WeekNumber, i, session); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSessionInput(weekNumber, index int, session entities.CreateSessionInput) error {
	if _, ok := validPlanSports[session.Sport]; !ok {
		return domainerrors.NewValidationError("week %d session %d: sport must be one of run, boulder, lead, strength, other, got %q", weekNumber, index, session.Sport)
	}
	if session.SessionType == "" {
		return domainerrors.NewValidationError("week %d session %d: sessionType is required", weekNumber, index)
	}
	if session.PlannedDate.IsZero() {
		return domainerrors.NewValidationError("week %d session %d: plannedDate is required", weekNumber, index)
	}
	if session.TargetDurationMin != nil && *session.TargetDurationMin < 0 {
		return domainerrors.NewValidationError("week %d session %d: targetDurationMin must be >= 0", weekNumber, index)
	}
	if session.TargetDistanceKm != nil && *session.TargetDistanceKm < 0 {
		return domainerrors.NewValidationError("week %d session %d: targetDistanceKm must be >= 0", weekNumber, index)
	}
	if len(session.Description) > 2000 {
		return domainerrors.NewValidationError("week %d session %d: description must be at most 2000 characters", weekNumber, index)
	}
	return nil
}

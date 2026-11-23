// Package goals implements goal business logic: catalog validation and the
// one-primary invariant. Multiple goals can be active, exactly one primary.
package goals

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
)

// Service holds the goal business logic dependencies.
type Service struct {
	repo *repository.Repository
}

// New builds a goals Service.
func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreateGoal validates and creates a goal. Creating a primary goal clears
// the primary flag of other active goals.
func (s *Service) CreateGoal(ctx context.Context, athleteID uuid.UUID, input entities.CreateGoalInput) (entities.Goal, error) {
	if err := validateGoalInput(&input); err != nil {
		return entities.Goal{}, err
	}

	var goal entities.Goal
	err := s.repo.WithTx(ctx, func(tx *repository.Repository) error {
		if input.IsPrimary {
			if err := tx.ClearPrimaryGoals(ctx, athleteID); err != nil {
				return err
			}
		}
		var err error
		goal, err = tx.CreateGoal(ctx, entities.Goal{
			AthleteID:    athleteID,
			ActivityType: input.ActivityType,
			TargetMetric: input.TargetMetric,
			TargetValue:  input.TargetValue,
			TargetDate:   input.TargetDate,
			IsPrimary:    input.IsPrimary,
			Notes:        input.Notes,
		})
		return err
	})
	if err != nil {
		return entities.Goal{}, err
	}
	return goal, nil
}

// ListGoals returns all goals of the athlete.
func (s *Service) ListGoals(ctx context.Context, athleteID uuid.UUID) ([]entities.Goal, error) {
	return s.repo.ListGoals(ctx, athleteID)
}

// GetGoal returns one goal scoped to the athlete.
func (s *Service) GetGoal(ctx context.Context, athleteID, goalID uuid.UUID) (entities.Goal, error) {
	goal, err := s.repo.GetGoal(ctx, athleteID, goalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Goal{}, domainerrors.NewNotFoundError("goal %s not found", goalID)
		}
		return entities.Goal{}, err
	}
	return goal, nil
}

// UpdateGoal applies a partial update. Setting is_primary clears the primary
// flag of other active goals.
func (s *Service) UpdateGoal(ctx context.Context, athleteID, goalID uuid.UUID, input entities.UpdateGoalInput) (entities.Goal, error) {
	if input.Status != nil && !validStatus(*input.Status) {
		return entities.Goal{}, domainerrors.NewValidationError("status must be one of active, achieved, abandoned, got %q", *input.Status)
	}

	if _, err := s.GetGoal(ctx, athleteID, goalID); err != nil {
		return entities.Goal{}, err
	}

	var updated entities.Goal
	err := s.repo.WithTx(ctx, func(tx *repository.Repository) error {
		if input.IsPrimary != nil && *input.IsPrimary {
			if err := tx.ClearPrimaryGoals(ctx, athleteID); err != nil {
				return err
			}
		}
		var err error
		updated, err = tx.UpdateGoal(ctx, athleteID, goalID, input)
		return err
	})
	if err != nil {
		return entities.Goal{}, err
	}
	return updated, nil
}

// DeleteGoal removes a goal.
func (s *Service) DeleteGoal(ctx context.Context, athleteID, goalID uuid.UUID) error {
	if err := s.repo.DeleteGoal(ctx, athleteID, goalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainerrors.NewNotFoundError("goal %s not found", goalID)
		}
		return err
	}
	return nil
}

func validateGoalInput(input *entities.CreateGoalInput) error {
	switch input.ActivityType {
	case entities.GoalActivityRun:
		if input.TargetMetric != entities.TargetMetricDistanceKm {
			return domainerrors.NewValidationError("run goals must use targetMetric %q, got %q", entities.TargetMetricDistanceKm, input.TargetMetric)
		}
		distance, err := strconv.ParseFloat(input.TargetValue, 64)
		if err != nil || distance <= 0 || distance > 500 {
			return domainerrors.NewValidationError("run goal targetValue must be a distance in km in (0, 500], got %q", input.TargetValue)
		}
	case entities.GoalActivityBoulder:
		if input.TargetMetric != entities.TargetMetricGrade {
			return domainerrors.NewValidationError("boulder goals must use targetMetric %q, got %q", entities.TargetMetricGrade, input.TargetMetric)
		}
		if !entities.ValidBoulderGrade(input.TargetValue) {
			return domainerrors.NewValidationError("boulder goal targetValue must be a Font boulder grade (e.g. 6B, 7A, 7C), got %q", input.TargetValue)
		}
	case entities.GoalActivityLead:
		if input.TargetMetric != entities.TargetMetricGrade {
			return domainerrors.NewValidationError("lead goals must use targetMetric %q, got %q", entities.TargetMetricGrade, input.TargetMetric)
		}
		if !entities.ValidLeadGrade(input.TargetValue) {
			return domainerrors.NewValidationError("lead goal targetValue must be a Font lead grade (e.g. 6a, 7a+), got %q", input.TargetValue)
		}
	default:
		return domainerrors.NewValidationError("activityType must be one of run, boulder, lead, got %q", input.ActivityType)
	}

	if input.TargetDate != nil && input.TargetDate.Time().Before(time.Now().AddDate(-1, 0, 0)) {
		return domainerrors.NewValidationError("targetDate cannot be more than one year in the past, got %s", input.TargetDate.String())
	}
	return nil
}

func validStatus(status entities.GoalStatus) bool {
	return status == entities.GoalStatusActive ||
		status == entities.GoalStatusAchieved ||
		status == entities.GoalStatusAbandoned
}

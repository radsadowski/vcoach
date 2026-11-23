package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const goalColumns = `id, athlete_id, activity_type, target_metric, target_value, target_date, is_primary, status, notes, created_at, updated_at`

// CreateGoal inserts a goal and returns the stored row.
func (r *Repository) CreateGoal(ctx context.Context, goal entities.Goal) (entities.Goal, error) {
	const q = `
		INSERT INTO goals (athlete_id, activity_type, target_metric, target_value, target_date, is_primary, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + goalColumns

	var created entities.Goal
	err := r.db.GetContext(ctx, &created, q,
		goal.AthleteID, goal.ActivityType, goal.TargetMetric, goal.TargetValue,
		goal.TargetDate, goal.IsPrimary, goal.Notes,
	)
	if err != nil {
		return entities.Goal{}, fmt.Errorf("insert goal: %w", err)
	}
	return created, nil
}

// GetGoal returns one goal scoped to the athlete.
func (r *Repository) GetGoal(ctx context.Context, athleteID, goalID uuid.UUID) (entities.Goal, error) {
	const q = `SELECT ` + goalColumns + ` FROM goals WHERE athlete_id = $1 AND id = $2`

	var goal entities.Goal
	if err := r.db.GetContext(ctx, &goal, q, athleteID, goalID); err != nil {
		return entities.Goal{}, err
	}
	return goal, nil
}

// ListGoals returns all goals of the athlete, newest first.
func (r *Repository) ListGoals(ctx context.Context, athleteID uuid.UUID) ([]entities.Goal, error) {
	const q = `SELECT ` + goalColumns + ` FROM goals WHERE athlete_id = $1 ORDER BY created_at DESC`

	var goals []entities.Goal
	if err := r.db.SelectContext(ctx, &goals, q, athleteID); err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	return goals, nil
}

// UpdateGoal applies a partial update and returns the stored row.
func (r *Repository) UpdateGoal(ctx context.Context, athleteID, goalID uuid.UUID, input entities.UpdateGoalInput) (entities.Goal, error) {
	const q = `
		UPDATE goals SET
			status = COALESCE($3, status),
			is_primary = COALESCE($4, is_primary),
			notes = COALESCE($5, notes),
			updated_at = now()
		WHERE athlete_id = $1 AND id = $2
		RETURNING ` + goalColumns

	var updated entities.Goal
	err := r.db.GetContext(ctx, &updated, q, athleteID, goalID, input.Status, input.IsPrimary, input.Notes)
	if err != nil {
		return entities.Goal{}, err
	}
	return updated, nil
}

// DeleteGoal removes a goal scoped to the athlete.
func (r *Repository) DeleteGoal(ctx context.Context, athleteID, goalID uuid.UUID) error {
	const q = `DELETE FROM goals WHERE athlete_id = $1 AND id = $2`

	res, err := r.db.ExecContext(ctx, q, athleteID, goalID)
	if err != nil {
		return fmt.Errorf("delete goal: %w", err)
	}
	return requireRowAffected(res, "goal")
}

// ClearPrimaryGoals unsets is_primary on all active goals of the athlete.
func (r *Repository) ClearPrimaryGoals(ctx context.Context, athleteID uuid.UUID) error {
	const q = `UPDATE goals SET is_primary = false, updated_at = now() WHERE athlete_id = $1 AND is_primary AND status = 'active'`

	if _, err := r.db.ExecContext(ctx, q, athleteID); err != nil {
		return fmt.Errorf("clear primary goals: %w", err)
	}
	return nil
}

// GoalExists reports whether the goal belongs to the athlete.
func (r *Repository) GoalExists(ctx context.Context, athleteID, goalID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM goals WHERE athlete_id = $1 AND id = $2)`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, q, athleteID, goalID); err != nil {
		return false, err
	}
	return exists, nil
}

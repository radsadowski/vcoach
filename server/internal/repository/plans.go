package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const planColumns = `id, athlete_id, goal_id, version, status, title, notes, created_at`
const planWeekColumns = `id, plan_id, week_number, start_date, focus`
const planSessionColumns = `id, week_id, athlete_id, sport, session_type, description, planned_date, target_duration_min, target_distance_km, target_intensity, status, matched_activity_id, completed_at`

// CreatePlanVersion atomically archives the current active plan (if any) and
// inserts the next plan version with its weeks and sessions. Plans are
// append-only: adaptation always creates a new version.
func (r *Repository) CreatePlanVersion(ctx context.Context, athleteID uuid.UUID, goalID *uuid.UUID, input entities.CreatePlanInput) (entities.PlanWithSessions, error) {
	var created entities.PlanWithSessions

	err := r.WithTx(ctx, func(tx *Repository) error {
		const archive = `UPDATE plans SET status = 'archived' WHERE athlete_id = $1 AND status = 'active'`
		if _, err := tx.db.ExecContext(ctx, archive, athleteID); err != nil {
			return fmt.Errorf("archive active plan: %w", err)
		}

		const nextVersion = `SELECT COALESCE(MAX(version), 0) + 1 FROM plans WHERE athlete_id = $1`
		var version int
		if err := tx.db.GetContext(ctx, &version, nextVersion, athleteID); err != nil {
			return fmt.Errorf("next plan version: %w", err)
		}

		const insertPlan = `INSERT INTO plans (athlete_id, goal_id, version, title, notes) VALUES ($1, $2, $3, $4, $5) RETURNING ` + planColumns
		if err := tx.db.GetContext(ctx, &created.Plan, insertPlan, athleteID, goalID, version, input.Title, input.Notes); err != nil {
			return fmt.Errorf("insert plan: %w", err)
		}

		for _, week := range input.Weeks {
			const insertWeek = `INSERT INTO plan_weeks (plan_id, week_number, start_date, focus) VALUES ($1, $2, $3, $4) RETURNING ` + planWeekColumns
			var storedWeek entities.PlanWeekWithSessions
			if err := tx.db.GetContext(ctx, &storedWeek.PlanWeek, insertWeek, created.ID, week.WeekNumber, week.StartDate, week.Focus); err != nil {
				return fmt.Errorf("insert plan week %d: %w", week.WeekNumber, err)
			}

			for _, session := range week.Sessions {
				const insertSession = `
					INSERT INTO plan_sessions (week_id, athlete_id, sport, session_type, description, planned_date, target_duration_min, target_distance_km, target_intensity)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
					RETURNING ` + planSessionColumns
				var storedSession entities.PlanSession
				err := tx.db.GetContext(ctx, &storedSession, insertSession,
					storedWeek.ID, athleteID, session.Sport, session.SessionType, session.Description,
					session.PlannedDate, session.TargetDurationMin, session.TargetDistanceKm, session.TargetIntensity,
				)
				if err != nil {
					return fmt.Errorf("insert plan session: %w", err)
				}
				storedWeek.Sessions = append(storedWeek.Sessions, storedSession)
			}
			created.Weeks = append(created.Weeks, storedWeek)
		}
		return nil
	})
	if err != nil {
		return entities.PlanWithSessions{}, err
	}
	return created, nil
}

// GetPlan returns the plan with weeks and sessions, scoped to the athlete.
func (r *Repository) GetPlan(ctx context.Context, athleteID, planID uuid.UUID) (entities.PlanWithSessions, error) {
	const q = `SELECT ` + planColumns + ` FROM plans WHERE athlete_id = $1 AND id = $2`

	var plan entities.PlanWithSessions
	if err := r.db.GetContext(ctx, &plan, q, athleteID, planID); err != nil {
		return entities.PlanWithSessions{}, err
	}

	var weeks []entities.PlanWeekWithSessions
	if err := r.db.SelectContext(ctx, &weeks, `SELECT `+planWeekColumns+` FROM plan_weeks WHERE plan_id = $1 ORDER BY week_number`, planID); err != nil {
		return entities.PlanWithSessions{}, fmt.Errorf("list plan weeks: %w", err)
	}

	sessionsByWeek := make(map[uuid.UUID][]entities.PlanSession)
	var sessions []entities.PlanSession
	if err := r.db.SelectContext(ctx, &sessions, `SELECT `+planSessionColumns+` FROM plan_sessions WHERE athlete_id = $1 AND week_id = ANY($2) ORDER BY planned_date`, athleteID, weekIDs(weeks)); err != nil {
		return entities.PlanWithSessions{}, fmt.Errorf("list plan sessions: %w", err)
	}
	for _, session := range sessions {
		sessionsByWeek[session.WeekID] = append(sessionsByWeek[session.WeekID], session)
	}
	for i := range weeks {
		weeks[i].Sessions = sessionsByWeek[weeks[i].ID]
		if weeks[i].Sessions == nil {
			weeks[i].Sessions = []entities.PlanSession{}
		}
	}

	plan.Weeks = weeks
	return plan, nil
}

// ListPlans returns all plan versions of the athlete, newest first.
func (r *Repository) ListPlans(ctx context.Context, athleteID uuid.UUID) ([]entities.Plan, error) {
	const q = `SELECT ` + planColumns + ` FROM plans WHERE athlete_id = $1 ORDER BY version DESC`

	var plans []entities.Plan
	if err := r.db.SelectContext(ctx, &plans, q, athleteID); err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	return plans, nil
}

// GetActivePlan returns the athlete's currently active plan, if any.
func (r *Repository) GetActivePlan(ctx context.Context, athleteID uuid.UUID) (*entities.Plan, error) {
	const q = `SELECT ` + planColumns + ` FROM plans WHERE athlete_id = $1 AND status = 'active'`

	var plan entities.Plan
	if err := r.db.GetContext(ctx, &plan, q, athleteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

// ArchivePlan archives a plan version, scoped to the athlete.
func (r *Repository) ArchivePlan(ctx context.Context, athleteID, planID uuid.UUID) error {
	const q = `UPDATE plans SET status = 'archived' WHERE athlete_id = $1 AND id = $2 AND status = 'active'`

	res, err := r.db.ExecContext(ctx, q, athleteID, planID)
	if err != nil {
		return fmt.Errorf("archive plan: %w", err)
	}
	return requireRowAffected(res, "plan")
}

func weekIDs(weeks []entities.PlanWeekWithSessions) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(weeks))
	for _, week := range weeks {
		ids = append(ids, week.ID)
	}
	return ids
}

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const climbingSessionColumns = `id, athlete_id, session_date, venue, discipline, hardest_sent, hardest_attempted, attempts, rpe, notes, created_at`

// CreateClimbingSession inserts a manually logged climbing session.
func (r *Repository) CreateClimbingSession(ctx context.Context, athleteID uuid.UUID, input entities.CreateClimbingSessionInput) (entities.ClimbingSession, error) {
	const q = `
		INSERT INTO climbing_sessions (athlete_id, session_date, venue, discipline, hardest_sent, hardest_attempted, attempts, rpe, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + climbingSessionColumns

	var created entities.ClimbingSession
	err := r.db.GetContext(ctx, &created, q,
		athleteID, input.SessionDate, input.Venue, input.Discipline,
		input.HardestSent, input.HardestAttempted, input.Attempts, input.RPE, input.Notes,
	)
	if err != nil {
		return entities.ClimbingSession{}, fmt.Errorf("insert climbing session: %w", err)
	}
	return created, nil
}

// GetClimbingSession returns one climbing session scoped to the athlete.
func (r *Repository) GetClimbingSession(ctx context.Context, athleteID, sessionID uuid.UUID) (entities.ClimbingSession, error) {
	const q = `SELECT ` + climbingSessionColumns + ` FROM climbing_sessions WHERE athlete_id = $1 AND id = $2`

	var session entities.ClimbingSession
	if err := r.db.GetContext(ctx, &session, q, athleteID, sessionID); err != nil {
		return entities.ClimbingSession{}, err
	}
	return session, nil
}

// ListClimbingSessions returns climbing sessions in the inclusive date range,
// newest first. Nil range bounds are open-ended.
func (r *Repository) ListClimbingSessions(ctx context.Context, athleteID uuid.UUID, from, to *time.Time) ([]entities.ClimbingSession, error) {
	const q = `
		SELECT ` + climbingSessionColumns + ` FROM climbing_sessions
		WHERE athlete_id = $1
		  AND ($2::date IS NULL OR session_date >= $2)
		  AND ($3::date IS NULL OR session_date <= $3)
		ORDER BY session_date DESC, created_at DESC`

	var sessions []entities.ClimbingSession
	if err := r.db.SelectContext(ctx, &sessions, q, athleteID, from, to); err != nil {
		return nil, fmt.Errorf("list climbing sessions: %w", err)
	}
	return sessions, nil
}

// UpdateClimbingSession applies a partial update and returns the stored row.
func (r *Repository) UpdateClimbingSession(ctx context.Context, athleteID, sessionID uuid.UUID, input entities.UpdateClimbingSessionInput) (entities.ClimbingSession, error) {
	const q = `
		UPDATE climbing_sessions SET
			session_date = COALESCE($3, session_date),
			venue = COALESCE($4, venue),
			discipline = COALESCE($5, discipline),
			hardest_sent = COALESCE($6, hardest_sent),
			hardest_attempted = COALESCE($7, hardest_attempted),
			attempts = COALESCE($8, attempts),
			rpe = COALESCE($9, rpe),
			notes = COALESCE($10, notes)
		WHERE athlete_id = $1 AND id = $2
		RETURNING ` + climbingSessionColumns

	var updated entities.ClimbingSession
	err := r.db.GetContext(ctx, &updated, q,
		athleteID, sessionID,
		input.SessionDate, input.Venue, input.Discipline,
		input.HardestSent, input.HardestAttempted, input.Attempts, input.RPE, input.Notes,
	)
	if err != nil {
		return entities.ClimbingSession{}, err
	}
	return updated, nil
}

// DeleteClimbingSession removes a climbing session scoped to the athlete.
func (r *Repository) DeleteClimbingSession(ctx context.Context, athleteID, sessionID uuid.UUID) error {
	const q = `DELETE FROM climbing_sessions WHERE athlete_id = $1 AND id = $2`

	res, err := r.db.ExecContext(ctx, q, athleteID, sessionID)
	if err != nil {
		return fmt.Errorf("delete climbing session: %w", err)
	}
	return requireRowAffected(res, "climbing session")
}
